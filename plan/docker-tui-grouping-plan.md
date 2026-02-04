# Kế hoạch & phân tích yêu cầu: gom nhóm container/docker trong TUI

## Phân tích yêu cầu
1. **Gom nhóm hiển thị container & docker theo VM**
   - TUI cần hiển thị cấu trúc dạng cây: VM → Docker host → Containers.
   - Hỗ trợ **collapse/expand** từng nhóm để giống cách VSCode nhóm containers.

2. **Tùy chọn thao tác khi chọn VM**
   - Khi focus vào VM, hiển thị menu hành động (ví dụ: control docker, vào bash trực tiếp).

3. **Tùy chọn thao tác khi chọn docker/container**
   - Khi focus vào docker host/nhóm container: cho phép **multi-select** và start/stop hàng loạt.
   - Khi focus vào từng container: hỗ trợ các hành động như restart/stop/down/etc.

4. **Hành vi khi vào docker**
   - Hiện tại vào thẳng bash và không quay lại được bằng Ctrl+C.
   - Cần cơ chế thoát/return về TUI (ví dụ: phím tắt thoát, hoặc mở terminal subsession có thể đóng).

5. **Search theo kiểu Vim**
   - Tại màn hình docker: hỗ trợ search (ví dụ nhấn `:` rồi gõ lệnh `:q keyword`).
   - `Esc` để thoát khỏi chế độ search.

## Kế hoạch thực hiện (đề xuất)
1. **Khảo sát cấu trúc TUI hiện tại (file & entry points)**
   - Đọc `internal/ui/ui.go` để hiểu `Model.Update/View`, input handling và cách gọi `tea.Exec`.
   - Đọc `internal/state/state.go` để nắm state machine hiện tại và xác định state cần bổ sung cho tree/mode/search.
   - Xác định nơi load VM/containers (`Model.Init` + `LoadContainers`) và mapping VM → containers.
   - Ghi chú các struct/type hiện có và luồng cập nhật state khi nhận input.

2. **Thiết kế lại model dữ liệu dạng cây (data/model)**
   - Tạo tree model: `VM node → Docker host node → Container node`.
   - Thêm state `expanded/collapsed` ở node group.
   - Đề xuất struct (ví dụ): `Node { id, kind, label, children, expanded, selected, meta }`.
   - `kind` là enum (`VM`, `DockerGroup`, `Container`) để quyết định action/menu.
   - `meta` giữ thông tin backend (vm index/id, container id, image, status).
   - Thêm file mới `internal/ui/tree.go` để chứa struct + helper:
     - `BuildTree(vms, containersByVM)` → []Node root.
     - `FlattenVisible(tree, filter)` → []Node view (theo expanded + search).
     - `ToggleExpanded(nodeID)` + `SetSelected(nodeID, bool)`.

3. **Cập nhật UI render (View)**
   - Chuyển `View()` từ list phẳng sang render tree `FlattenVisible(...)`.
   - Render theo tree với indent/marker.
   - Cho phép toggle collapse/expand bằng phím tắt (ví dụ Enter/Space).
   - Tùy chọn icon/marker: `▸/▾` cho collapsed/expanded.
   - Với multi-select: hiển thị prefix `[x]/[ ]` để biểu thị chọn.
   - Update footer status theo mode hiện tại: normal / action menu / search.

4. **Context actions theo node type (Update + actions)**
   - VM node: menu action (control docker / bash).
   - Docker group: bulk start/stop.
   - Container: start/stop/restart/down.
   - Hỗ trợ multi-select cho container nodes.
   - Gợi ý luồng: nhấn `a` để mở action menu theo node type.
   - Gắn handler theo node kind (VM/DockerGroup/Container).
   - Với multi-select: áp action lên danh sách node đã chọn.
   - Thêm struct `Action` trong `internal/ui/actions.go`:
     - `Label`, `Key`, `Run(cmd)`.
   - Hàm `ActionsForNode(node)` trả về danh sách action.
   - Trong `Update`: khi chọn action → gọi `m.app`/`docker` để thực thi.

5. **Cải thiện hành vi vào shell (TUI ↔ shell)**
   - Đổi flow: mở shell ở chế độ “subscreen” và có phím tắt thoát.
   - Hoặc xử lý Ctrl+C để quay lại TUI thay vì exit.
   - Ưu tiên mở `exec.Command` vào PTY và gắn keybinding `Ctrl+Q`/`Esc` để return.
   - Tách logic shell ra file `internal/ui/shell.go`:
     - `StartContainerShell(vm, containerID)` và `StartVMShell(vm)`.
     - Bắt `Esc/Ctrl+Q` để return về TUI (không thoát app).
   - Đảm bảo cleanup session khi thoát (đóng PTY, stop goroutine đọc output).

6. **Thêm search mode (command palette)**
   - Implement command palette kiểu Vim: nhấn `:` để vào command mode.
   - Parse lệnh dạng `q <keyword>` hoặc `:q keyword`.
   - `Esc` thoát search mode.
   - Khi search: lọc tree nodes theo label (case-insensitive) và highlight kết quả.
   - Khi thoát search: restore full tree và vị trí focus ban đầu.
   - Thêm field `commandMode bool`, `commandInput string` trong `Model`.
   - Update `Update`:
     - Khi `:` → bật commandMode.
     - Khi `Enter` trong commandMode → parse và apply filter.
     - Khi `Esc` → exit commandMode + clear filter.

7. **Kiểm thử UI/UX**
   - Test với VM có nhiều docker/containers.
   - Test collapse/expand, multi-select, search, shell exit.
   - Kiểm tra thao tác bulk start/stop trên nhiều container đã chọn.
   - Thử nghiệm chuyển mode: normal → action menu → search → back.

---

*File này chỉ mô tả kế hoạch. Sau khi xác nhận, sẽ triển khai code theo từng bước ở trên.*
