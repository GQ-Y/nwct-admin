import Cocoa
import Darwin
import FlutterMacOS

@main
class AppDelegate: FlutterAppDelegate {
  override init() {
    super.init()
    // 当通过 launchd/LaunchServices 启动时，stdout/stderr 可能是已关闭的 pipe；
    // 某些底层库写入会触发 SIGPIPE 导致进程直接退出。这里统一忽略 SIGPIPE。
    signal(SIGPIPE, SIG_IGN)
  }

  override func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
    // 保持常驻：由托盘“退出”来结束进程，避免窗口未打开/被关闭时直接退出。
    return false
  }

  override func applicationSupportsSecureRestorableState(_ app: NSApplication) -> Bool {
    return true
  }
}
