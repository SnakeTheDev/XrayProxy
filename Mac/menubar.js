// Home-Reality menu bar app (runs with macOS's built-in osascript - nothing to install)
ObjC.import('Cocoa');

var HOME = ObjC.unwrap($.NSHomeDirectory());
var DIR = HOME + '/.homereality';
var CTL = DIR + '/proxyctl.sh';
var NAME = 'Home-Reality';
var HOST = 'play.rowstonsoftware.com:443';
var LOCAL = '127.0.0.1:2080';

var sa = Application.currentApplication();
sa.includeStandardAdditions = true;

// Works whether the bridge exposes a no-argument method as a property or as a function.
function z(obj, name) {
  var m = obj[name];
  if (typeof m === 'function') { try { return m.call(obj); } catch (e) { return m; } }
  return m;
}
function sh(cmd) { try { return sa.doShellScript(cmd); } catch (e) { return ''; } }
function ctl(args, bg) { return sh('"' + CTL + '" ' + args + (bg ? ' >/dev/null 2>&1 &' : '')); }
function log(m) { sh('echo "$(date "+%F %T") ' + String(m).replace(/["$`\\]/g, ' ') + '" >> "' + DIR + '/menubar.log"'); }
function color(r, g, b) { return $.NSColor.colorWithSRGBRedGreenBlueAlpha(r / 255, g / 255, b / 255, 1); }

sh('tail -n 100 "' + DIR + '/menubar.log" > "' + DIR + '/menubar.log.tmp" 2>/dev/null && mv "' + DIR + '/menubar.log.tmp" "' + DIR + '/menubar.log"');

// ---- shield + bolt icon, drawn in code (falls back to a system symbol, then to text) ----
function drawIcon(on) {
  var size = 18, k = size / 64;
  function P(x, y) { return $.NSMakePoint(x * k, (64 - y) * k); }
  var img = z($.NSImage, 'alloc').initWithSize($.NSMakeSize(size, size));
  z(img, 'lockFocus');

  var shield = z($.NSBezierPath, 'bezierPath');
  shield.moveToPoint(P(32, 3));
  shield.lineToPoint(P(56, 11));
  shield.lineToPoint(P(56, 30));
  shield.curveToPointControlPoint1ControlPoint2(P(32, 61), P(56, 46), P(45, 57));
  shield.curveToPointControlPoint1ControlPoint2(P(8, 30), P(19, 57), P(8, 46));
  shield.lineToPoint(P(8, 11));
  z(shield, 'closePath');
  var c1 = on ? color(61, 242, 160) : color(122, 129, 146);
  var c2 = on ? color(0, 144, 224) : color(84, 90, 104);
  var grad = z($.NSGradient, 'alloc').initWithStartingColorEndingColor(c1, c2);
  grad.drawInBezierPathAngle(shield, -65);

  var bolt = z($.NSBezierPath, 'bezierPath');
  bolt.moveToPoint(P(37, 10));
  bolt.lineToPoint(P(20, 35));
  bolt.lineToPoint(P(31, 35));
  bolt.lineToPoint(P(27, 54));
  bolt.lineToPoint(P(45, 26));
  bolt.lineToPoint(P(34, 26));
  bolt.lineToPoint(P(40, 10));
  z(bolt, 'closePath');
  z(on ? color(255, 255, 255) : color(205, 210, 220), 'setFill');
  z(bolt, 'fill');

  z(img, 'unlockFocus');
  return img;
}
function makeIcon(on) {
  try { return drawIcon(on); } catch (e) { log('icon draw failed: ' + e); }
  try { return $.NSImage.imageWithSystemSymbolNameAccessibilityDescription(on ? 'checkmark.shield.fill' : 'shield', NAME); } catch (e2) { log('symbol icon failed: ' + e2); }
  return null;
}
var ICON_ON = makeIcon(true);
var ICON_OFF = makeIcon(false);

// ---- app + menu ----
var app = z($.NSApplication, 'sharedApplication');
app.setActivationPolicy($.NSApplicationActivationPolicyAccessory);

var state = { last: '', busyUntil: 0, ticks: 0 };

function onToggle() {
  if (Date.now() < state.busyUntil) return;
  var running = ctl('status') === 'on';
  state.busyUntil = Date.now() + (running ? 4000 : 8000);
  sh('touch "' + DIR + '/.busy"');
  ctl(running ? 'stop' : 'start', true);
  updateUI('busy');
}
function onQuit() {
  var st = ctl('status');
  if (st === 'on' || st === 'busy') {
    app.activateIgnoringOtherApps(true);
    try {
      var r = sa.displayDialog('Turn off the proxy before quitting?', {
        withTitle: NAME, buttons: ['Cancel', 'Leave on', 'Turn off'], defaultButton: 'Turn off', cancelButton: 'Cancel'
      });
      if (r.buttonReturned === 'Turn off') ctl('stop');
    } catch (e) { return; }
  }
  app.terminate(null);
}
function guard(name, fn) { return function (s) { try { fn(); } catch (e) { log(name + ' failed: ' + e); } }; }

ObjC.registerSubclass({
  name: 'HRDelegate',
  superclass: 'NSObject',
  methods: {
    'toggle:':      { types: ['void', ['id']], implementation: guard('toggle', onToggle) },
    'tick:':        { types: ['void', ['id']], implementation: guard('tick', function () { tick(); }) },
    'copyURI:':     { types: ['void', ['id']], implementation: guard('copyURI', function () { ctl('copyuri'); }) },
    'openLog:':     { types: ['void', ['id']], implementation: guard('openLog', function () { sh('open -a TextEdit "' + DIR + '/proxy_core.log" || true'); }) },
    'openFolder:':  { types: ['void', ['id']], implementation: guard('openFolder', function () { sh('open "' + DIR + '"'); }) },
    'selfTest:':    { types: ['void', ['id']], implementation: guard('selfTest', function () { sh('open -a Terminal "' + DIR + '/Test.command"'); }) },
    'toggleLogin:': { types: ['void', ['id']], implementation: guard('toggleLogin', function () {
        var on = ctl('login status') === '1'; ctl('login ' + (on ? 'off' : 'on')); miLogin.state = on ? 0 : 1; }) },
    'quit:':        { types: ['void', ['id']], implementation: guard('quit', onQuit) },
    'powerOff:':    { types: ['void', ['id']], implementation: guard('powerOff', function () { ctl('stop'); }) }
  }
});
var delegate = z(z($.HRDelegate, 'alloc'), 'init');

var statusItem = z($.NSStatusBar, 'systemStatusBar').statusItemWithLength($.NSVariableStatusItemLength);
var button = z(statusItem, 'button');
var menu = z($.NSMenu, 'alloc').initWithTitle(NAME);
menu.autoenablesItems = false;

function addItem(title, sel, enabled) {
  var it = z($.NSMenuItem, 'alloc').initWithTitleActionKeyEquivalent(title, sel || '', '');
  if (sel) it.target = delegate;
  it.enabled = enabled !== false;
  menu.addItem(it);
  return it;
}
function addSep() { menu.addItem(z($.NSMenuItem, 'separatorItem')); }

var miStatus = addItem('○ Disconnected', null, false);
var miToggle = addItem('Turn On', 'toggle:');
var miError  = addItem('⚠ Couldn’t connect — open Log', 'openLog:');
miError.hidden = true;
addSep();
addItem('Host:  ' + HOST, null, false);
addItem('Local: ' + LOCAL, null, false);
addSep();
addItem('Copy URI', 'copyURI:');
addItem('Open Log', 'openLog:');
addItem('Open Folder', 'openFolder:');
addItem('Run Self-Test…', 'selfTest:');
addSep();
var miLogin = addItem('Start at Login', 'toggleLogin:');
miLogin.state = (ctl('login status') === '1') ? 1 : 0;
addItem('Quit', 'quit:');
statusItem.menu = menu;

function setIcon(img, tip, fallbackText) {
  if (button) {
    button.toolTip = tip;
    if (img) { button.image = img; button.title = ''; } else { button.title = fallbackText; }
  } else if (img) { statusItem.image = img; } else { statusItem.title = fallbackText; }
}

function updateUI(st) {
  if (st === state.last) return;
  state.last = st;
  miError.hidden = (st !== 'err');
  if (st === 'err') {
    var why = sh('tail -n 1 "' + DIR + '/last_error" | cut -c1-90');
    miError.title = '⚠ ' + (why || 'Couldn’t connect') + ' — open Log';
  }
  if (st === 'on') {
    miStatus.title = '● Connected'; miToggle.title = 'Turn Off'; miToggle.enabled = true;
    setIcon(ICON_ON, NAME + ' — Connected', '● HR');
  } else if (st === 'busy') {
    miStatus.title = '◐ Working…'; miToggle.enabled = false;
  } else {
    miStatus.title = '○ Disconnected'; miToggle.title = 'Turn On'; miToggle.enabled = true;
    setIcon(ICON_OFF, NAME + ' — Disconnected', '○ HR');
  }
}

function tick() {
  var st = ctl('status') || 'off';
  updateUI(st);
  state.ticks++;
  if (state.ticks % 4 === 1) sh('touch "' + DIR + '/menubar.alive"');   // heartbeat (used by the self-test)
  // never leave the Mac pointing at a dead proxy (e.g. after a crash or power loss)
  if ((st === 'off' || st === 'err') && state.ticks % 4 === 0 && ctl('stale') === '1') ctl('cleanup', true);
}

setIcon(ICON_OFF, NAME + ' — Disconnected', '○ HR');
$.NSTimer.scheduledTimerWithTimeIntervalTargetSelectorUserInfoRepeats(1.5, delegate, 'tick:', null, true);

// turn the proxy off automatically when the Mac shuts down / restarts / logs out
var nc = z($.NSWorkspace, 'sharedWorkspace').notificationCenter;
nc.addObserverSelectorNameObject(delegate, 'powerOff:', $.NSWorkspaceWillPowerOffNotification, null);

log('started; statusItem=' + (statusItem ? 'ok' : 'null') + ' button=' + (button ? 'ok' : 'null') + ' iconOn=' + (ICON_ON ? 'ok' : 'none') + ' iconOff=' + (ICON_OFF ? 'ok' : 'none'));
tick();
z(app, 'run');
