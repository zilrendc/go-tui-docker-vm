# Kế hoạch triển khai TUI SSH Docker

## 1. Mục tiêu & phạm vi
- Xây dựng TUI app bằng Go (BubbleTea) để quản lý nhiều VM Linux qua SSH và thao tác Docker CLI.
- Không dùng web app, không dùng Docker API.
- Đọc cấu hình VM từ JSON (password plaintext), không prompt nhập mật khẩu.

## 2. Kiến trúc tổng thể (theo quyết định)
- **TUI (BubbleTea)**: VM selector, container list, terminal view, keybindings.
- **Application Core**: state machine, SSH manager, Docker controller, config loader.
- **VM**: Linux + Docker Engine.

## 3. Cấu trúc thư mục dự kiến
```
tui-ssh-docker/
├── cmd/app/main.go
├── internal/
│   ├── app/
│   ├── state/
│   ├── ssh/
│   ├── docker/
│   ├── vm/
│   │   └── config.go
│   └── ui/
├── config/
│   └── vms.json
└── go.mod
```
- Không dùng thư mục `credential/`.

## 4. Thiết kế module & nhiệm vụ chính

### 4.1. `internal/vm/config.go`
- Định nghĩa `VMConfig`:
  ```go
  type VMConfig struct {
      ID       string `json:"id"`
      Name     string `json:"name"`
      Host     string `json:"host"`
      Port     int    `json:"port"`
      User     string `json:"user"`
      Password string `json:"password"`
  }
  ```
- Hàm load JSON từ `config/vms.json`:
  - Validate các trường bắt buộc (id, host, user, password, port > 0).
  - Trả về danh sách VM trong memory.

### 4.2. `internal/ssh/manager.go`
- `SSHManager` quản lý map `vmID -> *ssh.Client`.
- Kết nối bằng password từ JSON:
  ```go
  ssh.Password(vm.Password)
  ```
- Tái sử dụng client, tự reconnect khi timeout/err.
- Timeout mặc định 10s, `HostKeyCallback: ssh.InsecureIgnoreHostKey()`.

### 4.3. `internal/docker/controller.go`
- Thực thi Docker CLI qua SSH session:
  - `docker ps` để list container.
  - `docker exec -it <container> bash` để vào container.
- Trả kết quả về cho UI.

### 4.4. `internal/state/`
- State machine theo sơ đồ:
  - INIT → LOAD_CONFIG → VM_SELECTOR → CONNECTING_VM → CONTAINER_LIST → VM_SHELL/CONTAINER_SHELL → ERROR.
- Mỗi state có handler rõ ràng, chuyển state qua event.

### 4.5. `internal/ui/` (BubbleTea)
- View: VM selector, container list, terminal view.
- Keybindings cơ bản:
  - Up/Down: chọn VM/container
  - Enter: connect / exec
  - Esc/Back: quay lại
  - q: quit

### 4.6. `cmd/app/main.go`
- Khởi tạo app core + UI.
- Load config và boot state machine.

## 5. Dữ liệu cấu hình mẫu
- Tạo `config/vms.json` theo schema đã chốt.
- Ghi chú: file phải đặt quyền đọc phù hợp (600 nếu có thể).

## 6. Luồng UX
1. Start app
2. Load `vms.json`
3. VM Selector
4. Connect bằng password
5. Container list
6. Exec bash (VM hoặc container)

## 7. Build & distribution
- Build Windows binary:
  ```bash
  CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-s -w"
  ```

## 8. Rủi ro & lưu ý
- Password plaintext → cần bảo vệ quyền đọc file.
- Mất kết nối SSH → cần cơ chế reconnect và báo lỗi rõ ràng.

## 9. Các bước triển khai đề xuất (milestones)
1. Khởi tạo project Go + cấu trúc thư mục.
2. Implement config loader + validate JSON.
3. Implement SSH manager (connect/reuse/reconnect).
4. Implement Docker controller (list/exec).
5. Implement state machine.
6. Implement UI (BubbleTea views + keybindings).
7. Integrate end-to-end flow.
8. Build & manual test trên Windows.

---

# Hướng dẫn triển khai chi tiết theo từng bước (cho LLM coder)

> Mục tiêu: cung cấp interface, logic input/output, và luồng xử lý rõ ràng để dễ implement.

## Step 1: Khởi tạo project Go + cấu trúc thư mục
**Mục tiêu**
- Tạo skeleton project có thể build/run.

**Interface / Output**
- `go.mod` với module name (ví dụ `tui-ssh-docker`).
- `cmd/app/main.go` chạy được (tạm in “booting…”).
- Thư mục `internal/` tạo đúng cấu trúc.

**Logic**
- Khởi tạo module: `go mod init <module-name>`.
- Tạo file `main.go` với `func main()` khởi chạy app core (placeholder).

**Done khi**
- `go build ./cmd/app` chạy không lỗi.

---

## Step 2: Config loader + validate JSON
**Mục tiêu**
- Load file `config/vms.json` vào memory, validate field.

**Interface**
- File: `internal/vm/config.go`
- Types:
  ```go
  type VMConfig struct {
      ID       string `json:"id"`
      Name     string `json:"name"`
      Host     string `json:"host"`
      Port     int    `json:"port"`
      User     string `json:"user"`
      Password string `json:"password"`
  }

  type VMConfigFile struct {
      VMs []VMConfig `json:"vms"`
  }
  ```
- Functions:
  ```go
  func LoadVMConfig(path string) ([]VMConfig, error)
  ```

**Input/Output**
- Input: `path` string (ví dụ `config/vms.json`).
- Output: slice `[]VMConfig` hoặc error.

**Logic**
- Đọc file → unmarshal JSON → validate:
  - `id`, `name`, `host`, `user`, `password` không rỗng.
  - `port > 0`.
- Error rõ ràng: field nào thiếu/invalid.

**Done khi**
- Load sample JSON ok, validate fail khi thiếu field.

---

## Step 3: SSH Manager (connect/reuse/reconnect)
**Mục tiêu**
- Quản lý kết nối SSH cho nhiều VM, reuse client, reconnect khi lỗi.

**Interface**
- File: `internal/ssh/manager.go`
- Types:
  ```go
  type SSHManager struct {
      mu      sync.Mutex
      clients map[string]*ssh.Client
  }

  func NewSSHManager() *SSHManager
  func (m *SSHManager) Connect(vm VMConfig) (*ssh.Client, error)
  func (m *SSHManager) GetClient(vmID string) (*ssh.Client, bool)
  func (m *SSHManager) Close(vmID string)
  ```

**Input/Output**
- Input: `VMConfig`.
- Output: `*ssh.Client` hoặc error.

**Logic**
- Nếu đã có client và còn sống → return.
- Nếu client nil/đã lỗi → reconnect:
  - `ssh.ClientConfig` với `ssh.Password(vm.Password)`.
  - `HostKeyCallback: ssh.InsecureIgnoreHostKey()`.
  - `Timeout: 10s`.
- Store vào map.

**Done khi**
- Kết nối lại sau khi `client.Close()`.

---

## Step 4: Docker Controller (exec docker CLI qua SSH)
**Mục tiêu**
- Chạy lệnh Docker qua SSH session.

**Interface**
- File: `internal/docker/controller.go`
- Types:
  ```go
  type DockerController struct {
      ssh *SSHManager
  }

  func NewDockerController(ssh *SSHManager) *DockerController
  func (d *DockerController) ListContainers(vm VMConfig) ([]ContainerInfo, error)
  func (d *DockerController) ExecContainerShell(vm VMConfig, containerID string) error
  ```
- ContainerInfo model:
  ```go
  type ContainerInfo struct {
      ID    string
      Name  string
      Image string
      State string
  }
  ```

**Input/Output**
- Input: `VMConfig`, containerID.
- Output: list containers hoặc error.

**Logic**
- `ListContainers`: SSH session run
  - `docker ps --format "{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.State}}"`.
  - Parse output line-by-line → `ContainerInfo`.
- `ExecContainerShell`: run `docker exec -it <id> bash` trong PTY session.

**Done khi**
- `ListContainers` trả đúng data, parse được nhiều container.

---

## Step 5: State Machine
**Mục tiêu**
- Điều phối UI flow theo các state đã định.

**Interface**
- File: `internal/state/state.go`
- Types:
  ```go
  type State string
  const (
      StateInit State = "INIT"
      StateLoadConfig State = "LOAD_CONFIG"
      StateVMSelector State = "VM_SELECTOR"
      StateConnectingVM State = "CONNECTING_VM"
      StateContainerList State = "CONTAINER_LIST"
      StateVMShell State = "VM_SHELL"
      StateContainerShell State = "CONTAINER_SHELL"
      StateError State = "ERROR"
  )

  type StateMachine struct {
      Current State
      Err     error
  }

  func (sm *StateMachine) Transition(next State)
  ```

**Input/Output**
- Input: event từ UI (select VM, connect, list containers).
- Output: cập nhật state + error nếu có.

**Logic**
- INIT → LOAD_CONFIG tự động.
- Nếu load config fail → ERROR.
- VM selected → CONNECTING_VM → (success) CONTAINER_LIST.
- From CONTAINER_LIST:
  - Enter container → CONTAINER_SHELL.
  - VM shell → VM_SHELL.

**Done khi**
- State chuyển đúng và dễ hook vào UI.

---

## Step 6: UI (BubbleTea)
**Mục tiêu**
- Tạo UI cho VM selector, container list, terminal view.

**Interface**
- File: `internal/ui/model.go`
- Types:
  ```go
  type Model struct {
      state State
      vms []VMConfig
      containers []ContainerInfo
      selectedVM int
      selectedContainer int
      err error
  }
  ```
- Methods:
  ```go
  func (m Model) Init() tea.Cmd
  func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)
  func (m Model) View() string
  ```

**Input/Output**
- Input: keyboard events (Up/Down/Enter/Esc/q).
- Output: View string render TUI.

**Logic**
- `View()` switch theo state:
  - VM_SELECTOR: list VM.
  - CONTAINER_LIST: list container.
  - VM_SHELL/CONTAINER_SHELL: show terminal panel.
  - ERROR: show error + retry.
- `Update()`:
  - Up/Down thay đổi selection index.
  - Enter: trigger connect/list/exec.
  - Esc: back.
  - q: quit.

**Done khi**
- UI hiển thị list và điều hướng ok.

---

## Step 7: Integration
**Mục tiêu**
- Kết nối UI + state machine + ssh/docker.

**Interface**
- File: `internal/app/app.go`
- Types:
  ```go
  type App struct {
      ssh *SSHManager
      docker *DockerController
      sm *StateMachine
      vms []VMConfig
  }

  func NewApp() (*App, error)
  func (a *App) LoadConfig(path string) error
  func (a *App) ConnectVM(index int) error
  func (a *App) LoadContainers(vmIndex int) ([]ContainerInfo, error)
  ```

**Logic**
- `NewApp`: init managers.
- `LoadConfig`: gọi LoadVMConfig.
- `ConnectVM`: SSH connect, update state.
- `LoadContainers`: Docker list.

**Done khi**
- TUI flow chạy end-to-end.

---

## Step 8: Build & Manual Test
**Mục tiêu**
- Build được binary Windows và test cơ bản.

**Commands**
```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-s -w"
```

**Checklist**
- Mở app → thấy VM list.
- Kết nối VM → list container.
- Exec bash vào container/VM.
