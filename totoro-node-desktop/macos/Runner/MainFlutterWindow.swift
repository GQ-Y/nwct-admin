import Cocoa
import FlutterMacOS

class MainFlutterWindow: NSWindow {
  override func awakeFromNib() {
    let flutterViewController = FlutterViewController()
    let windowFrame = self.frame
    self.contentViewController = flutterViewController
    self.setFrame(windowFrame, display: true)

    RegisterGeneratedPlugins(registry: flutterViewController)

    // 更像“一个整体”的窗口：隐藏标题、标题栏透明，让 Flutter 背景延伸到顶部
    self.titleVisibility = .hidden
    self.titlebarAppearsTransparent = true
    self.styleMask.insert(.fullSizeContentView)
    self.isMovableByWindowBackground = true

    // 窗口最小尺寸（避免布局被压缩到不可用/触发布局异常）
    self.minSize = NSSize(width: 1040, height: 700)

    super.awakeFromNib()
  }
}
