import Cocoa
import FlutterMacOS

final class StatusItemController: NSObject {
  static let shared = StatusItemController()

  private var statusItem: NSStatusItem?
  private var inited = false

  func ensureSetup() {
    if inited { return }
    inited = true

    let item = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
    statusItem = item

    if let button = item.button {
      button.image = loadTrayImage()
      button.image?.isTemplate = true
      button.toolTip = "Totoro"
    }

    let menu = NSMenu()
    let openItem = NSMenuItem(title: "打开", action: #selector(showMainWindow), keyEquivalent: "")
    openItem.target = self
    menu.addItem(openItem)
    menu.addItem(NSMenuItem.separator())
    let quitItem = NSMenuItem(title: "退出", action: #selector(quitApp), keyEquivalent: "q")
    quitItem.target = self
    menu.addItem(quitItem)
    item.menu = menu
  }

  private func loadTrayImage() -> NSImage? {
    // Flutter assets 位于 Resources/flutter_assets 下
    if let res = Bundle.main.resourceURL {
      let url = res.appendingPathComponent("flutter_assets/assets/tray/tray.png")
      if let img = NSImage(contentsOf: url) {
        return img
      }
    }
    // fallback
    return NSImage(named: NSImage.statusAvailableName)
  }

  @objc private func showMainWindow() {
    // 找到 Flutter 主窗口并前置
    if let w = NSApp.windows.first(where: { $0 is MainFlutterWindow }) ?? NSApp.windows.first {
      w.makeKeyAndOrderFront(nil)
      NSApp.activate(ignoringOtherApps: true)
    }
  }

  @objc private func quitApp() {
    NSApp.terminate(nil)
  }
}

class MainFlutterWindow: NSWindow {
  override func awakeFromNib() {
    let flutterViewController = FlutterViewController()
    let windowFrame = self.frame
    self.contentViewController = flutterViewController
    self.setFrame(windowFrame, display: true)

    RegisterGeneratedPlugins(registry: flutterViewController)

    // System metrics channel (network bytes)
    let sysChannel = FlutterMethodChannel(name: "totoro/system", binaryMessenger: flutterViewController.engine.binaryMessenger)
    sysChannel.setMethodCallHandler({ (call: FlutterMethodCall, result: @escaping FlutterResult) -> Void in
      if call.method == "getNetworkBytes" {
        let v = getNetworkBytes()
        result(["rx": v.rx, "tx": v.tx])
        return
      }
      result(FlutterMethodNotImplemented)
    })

    // 初始化原生托盘（仅图标 + 打开/退出）
    StatusItemController.shared.ensureSetup()

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

private struct NetworkBytesValue {
  let rx: Int64
  let tx: Int64
}

private func getNetworkBytes() -> NetworkBytesValue {
  var rx: Int64 = 0
  var tx: Int64 = 0

  var ifaddr: UnsafeMutablePointer<ifaddrs>? = nil
  if getifaddrs(&ifaddr) != 0 {
    return NetworkBytesValue(rx: 0, tx: 0)
  }
  defer { freeifaddrs(ifaddr) }

  var ptr = ifaddr
  while ptr != nil {
    guard let ifa = ptr?.pointee else { break }
    let flags = Int32(ifa.ifa_flags)
    let isUp = (flags & IFF_UP) != 0
    let isLoopback = (flags & IFF_LOOPBACK) != 0
    if isUp && !isLoopback, let data = ifa.ifa_data {
      let ifData = data.assumingMemoryBound(to: if_data.self).pointee
      rx &+= Int64(ifData.ifi_ibytes)
      tx &+= Int64(ifData.ifi_obytes)
    }
    ptr = ifa.ifa_next
  }

  return NetworkBytesValue(rx: rx, tx: tx)
}
