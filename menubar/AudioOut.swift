// audio-out: the Mac's sound outputs, for the Kindle's music screen.
//
//   audio-out list           JSON: [{id, name, kind, current, connected}]
//   audio-out set <id>       make that output the Mac's sound output
//   audio-out connect <id>   connect paired Bluetooth headphones ("bt:<address>"), then use them
//
// Outputs are what macOS itself offers (MacBook speakers, AirPods when connected, a monitor…),
// plus paired Bluetooth headphones that aren't connected right now (connected: false).
// Spotify plays on "this computer", so it follows whatever is chosen here.

import CoreAudio
import Foundation
import IOBluetooth

struct Output: Codable {
    let id: String, name: String, kind: String
    let current: Bool, connected: Bool
}

func prop<T: BitwiseCopyable>(_ obj: AudioObjectID, _ sel: AudioObjectPropertySelector, _ scope: AudioObjectPropertyScope = kAudioObjectPropertyScopeGlobal, _ value: inout T) -> Bool {
    var addr = AudioObjectPropertyAddress(mSelector: sel, mScope: scope, mElement: kAudioObjectPropertyElementMain)
    var size = UInt32(MemoryLayout<T>.size)
    return AudioObjectGetPropertyData(obj, &addr, 0, nil, &size, &value) == noErr
}

func stringProp(_ obj: AudioObjectID, _ sel: AudioObjectPropertySelector) -> String {
    var s: Unmanaged<CFString>?
    var addr = AudioObjectPropertyAddress(mSelector: sel, mScope: kAudioObjectPropertyScopeGlobal, mElement: kAudioObjectPropertyElementMain)
    var size = UInt32(MemoryLayout<Unmanaged<CFString>?>.size)
    guard AudioObjectGetPropertyData(obj, &addr, 0, nil, &size, &s) == noErr, let v = s else { return "" }
    return v.takeRetainedValue() as String
}

func devices() -> [AudioObjectID] {
    var addr = AudioObjectPropertyAddress(mSelector: kAudioHardwarePropertyDevices, mScope: kAudioObjectPropertyScopeGlobal, mElement: kAudioObjectPropertyElementMain)
    var size: UInt32 = 0
    AudioObjectGetPropertyDataSize(AudioObjectID(kAudioObjectSystemObject), &addr, 0, nil, &size)
    var ids = [AudioObjectID](repeating: 0, count: Int(size) / MemoryLayout<AudioObjectID>.size)
    AudioObjectGetPropertyData(AudioObjectID(kAudioObjectSystemObject), &addr, 0, nil, &size, &ids)
    return ids
}

func hasOutput(_ d: AudioObjectID) -> Bool {
    var addr = AudioObjectPropertyAddress(mSelector: kAudioDevicePropertyStreamConfiguration, mScope: kAudioDevicePropertyScopeOutput, mElement: kAudioObjectPropertyElementMain)
    var size: UInt32 = 0
    guard AudioObjectGetPropertyDataSize(d, &addr, 0, nil, &size) == noErr, size > 0 else { return false }
    let buf = UnsafeMutableRawPointer.allocate(byteCount: Int(size), alignment: MemoryLayout<AudioBufferList>.alignment)
    defer { buf.deallocate() }
    guard AudioObjectGetPropertyData(d, &addr, 0, nil, &size, buf) == noErr else { return false }
    let list = UnsafeMutableAudioBufferListPointer(buf.assumingMemoryBound(to: AudioBufferList.self))
    return list.contains { $0.mNumberChannels > 0 }
}

func defaultOutput() -> AudioObjectID {
    var id = AudioObjectID(0)
    _ = prop(AudioObjectID(kAudioObjectSystemObject), kAudioHardwarePropertyDefaultOutputDevice, kAudioObjectPropertyScopeGlobal, &id)
    return id
}

func kind(_ d: AudioObjectID, _ name: String) -> String? {
    var t = UInt32(0)
    _ = prop(d, kAudioDevicePropertyTransportType, kAudioObjectPropertyScopeGlobal, &t)
    switch t {
    case kAudioDeviceTransportTypeVirtual, kAudioDeviceTransportTypeAggregate: return nil // Zoom, Teams, BlackHole…
    case kAudioDeviceTransportTypeBluetooth, kAudioDeviceTransportTypeBluetoothLE:
        let n = name.lowercased()
        return n.contains("airpods") || n.contains("headphone") || n.contains("buds") ? "headphones" : "speaker"
    case kAudioDeviceTransportTypeBuiltIn: return name.lowercased().contains("headphone") ? "headphones" : "laptop"
    case kAudioDeviceTransportTypeHDMI, kAudioDeviceTransportTypeDisplayPort: return "display"
    case kAudioDeviceTransportTypeAirPlay: return "speaker"
    default: return "speaker"
    }
}

func list(bluetooth: Bool = true) -> [Output] {
    let cur = defaultOutput()
    var out: [Output] = []
    var names = Set<String>()
    for d in devices() where hasOutput(d) {
        let name = stringProp(d, kAudioObjectPropertyName)
        guard let k = kind(d, name) else { continue }
        let uid = stringProp(d, kAudioDevicePropertyDeviceUID)
        out.append(Output(id: uid, name: name, kind: k, current: d == cur, connected: true))
        names.insert(name)
        if uid.hasSuffix(":output"), uid.count == 24 { // a Bluetooth output: "AA-BB-CC-11-22-33:output"
            remember(String(uid.prefix(17)).uppercased())
        }
    }
    // Paired Bluetooth headphones/speakers that aren't connected (e.g. AirPods in their case or on the phone).
    // (Needs the Bluetooth permission; "list nobt" skips this.)
    guard bluetooth else { return out }
    // Only ones that have played sound from this Mac before (remembered below), so the
    // list isn't every pair of headphones the Mac ever paired with or saw nearby.
    let known = seenBluetooth()
    var spare: [Output] = []
    for case let b as IOBluetoothDevice in (IOBluetoothDevice.pairedDevices() ?? []) {
        guard b.deviceClassMajor == kBluetoothDeviceClassMajorAudio, !b.isConnected(), let name = b.name, !names.contains(name),
              let addr = b.addressString?.uppercased(), known.contains(addr) else { continue }
        let n = name.lowercased()
        spare.append(Output(id: "bt:" + addr, name: name, kind: n.contains("airpods") || n.contains("headphone") || n.contains("buds") || n.contains("beats") ? "headphones" : "speaker", current: false, connected: false))
    }
    out += spare.sorted { $0.name < $1.name }.prefix(3)
    return out
}

// Bluetooth outputs this Mac has played through, kept in Application Support.
let seenFile = FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent("Library/Application Support/DeskBuddy/bluetooth-outputs.json")

func seenBluetooth() -> Set<String> {
    guard let d = try? Data(contentsOf: seenFile), let a = try? JSONDecoder().decode([String].self, from: d) else { return [] }
    return Set(a)
}

func remember(_ addr: String) {
    var s = seenBluetooth()
    guard !s.contains(addr) else { return }
    s.insert(addr)
    try? FileManager.default.createDirectory(at: seenFile.deletingLastPathComponent(), withIntermediateDirectories: true)
    try? JSONEncoder().encode(Array(s).sorted()).write(to: seenFile)
}

func setOutput(_ uid: String) -> Bool {
    for d in devices() where hasOutput(d) && stringProp(d, kAudioDevicePropertyDeviceUID) == uid {
        var id = d
        let size = UInt32(MemoryLayout<AudioObjectID>.size)
        var a1 = AudioObjectPropertyAddress(mSelector: kAudioHardwarePropertyDefaultOutputDevice, mScope: kAudioObjectPropertyScopeGlobal, mElement: kAudioObjectPropertyElementMain)
        var a2 = AudioObjectPropertyAddress(mSelector: kAudioHardwarePropertyDefaultSystemOutputDevice, mScope: kAudioObjectPropertyScopeGlobal, mElement: kAudioObjectPropertyElementMain)
        let ok = AudioObjectSetPropertyData(AudioObjectID(kAudioObjectSystemObject), &a1, 0, nil, size, &id) == noErr
        _ = AudioObjectSetPropertyData(AudioObjectID(kAudioObjectSystemObject), &a2, 0, nil, size, &id) // alert sounds follow too
        return ok
    }
    return false
}

// connect: bring paired Bluetooth headphones over, wait for macOS to list them, then use them.
func connect(_ id: String) -> Bool {
    let addr = String(id.dropFirst(3))
    guard let b = IOBluetoothDevice(addressString: addr) else { return false }
    let name = b.name ?? ""
    if !b.isConnected() && b.openConnection() != kIOReturnSuccess { return false }
    for _ in 0..<40 { // up to ~10 s for the audio device to appear
        for d in devices() where hasOutput(d) && stringProp(d, kAudioObjectPropertyName) == name {
            return setOutput(stringProp(d, kAudioDevicePropertyDeviceUID))
        }
        Thread.sleep(forTimeInterval: 0.25)
    }
    return false
}

let args = CommandLine.arguments
switch args.count > 1 ? args[1] : "" {
case "list":
    let data = try! JSONEncoder().encode(list(bluetooth: !(args.count > 2 && args[2] == "nobt")))
    print(String(data: data, encoding: .utf8)!)
case "set" where args.count > 2:
    exit(setOutput(args[2]) ? 0 : 1)
case "connect" where args.count > 2:
    exit(connect(args[2]) ? 0 : 1)
default:
    FileHandle.standardError.write("usage: audio-out list | set <id> | connect bt:<address>\n".data(using: .utf8)!)
    exit(2)
}
