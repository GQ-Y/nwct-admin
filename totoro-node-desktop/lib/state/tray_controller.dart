class TrayController {
  TrayController();

  /// macOS 托盘由原生 Swift 实现（避免 tray_manager 引发 AppKit 菜单崩溃）。
  /// Windows 托盘暂不启用；后续如需再补回。
  Future<void> init() async {}

  Future<void> dispose() async {}
}
