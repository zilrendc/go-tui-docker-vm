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
1. **Khảo sát cấu trúc TUI hiện tại**
   - Đọc code rendering và data model: xem cách hiển thị VM/docker/container.
   - Xác định nơi xây dựng danh sách hiển thị và nơi xử lý phím tắt.

2. **Thiết kế lại model dữ liệu dạng cây**
   - Tạo tree model: VM node → Docker host node → Container node.
   - Thêm state `expanded/collapsed` ở node group.

3. **Cập nhật UI render**
   - Render theo tree với indent/marker.
   - Cho phép toggle collapse/expand bằng phím tắt (ví dụ Enter/Space).

4. **Context actions theo node type**
   - VM node: menu action (control docker / bash).
   - Docker group: bulk start/stop.
   - Container: start/stop/restart/down.
   - Hỗ trợ multi-select cho container nodes.

5. **Cải thiện hành vi vào shell**
   - Đổi flow: mở shell ở chế độ “subscreen” và có phím tắt thoát.
   - Hoặc xử lý Ctrl+C để quay lại TUI thay vì exit.

6. **Thêm search mode**
   - Implement command palette kiểu Vim: nhấn `:` để vào command mode.
   - Parse lệnh dạng `q <keyword>` hoặc `:q keyword`.
   - `Esc` thoát search mode.

7. **Kiểm thử UI/UX**
   - Test với VM có nhiều docker/containers.
   - Test collapse/expand, multi-select, search, shell exit.

---

*File này chỉ mô tả kế hoạch. Sau khi xác nhận, sẽ triển khai code theo từng bước ở trên.*
