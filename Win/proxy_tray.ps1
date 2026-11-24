# ---- sharp rendering on high-DPI screens (must happen before any window exists) ----
Add-Type -Namespace Native -Name Api -MemberDefinition @'
[DllImport("user32.dll")] public static extern bool SetProcessDPIAware();
[DllImport("wininet.dll")] public static extern bool InternetSetOption(IntPtr h, int o, IntPtr b, int l);
[DllImport("dwmapi.dll")] public static extern int DwmSetWindowAttribute(IntPtr hwnd, int attr, ref int val, int size);
'@
[Native.Api]::SetProcessDPIAware() | Out-Null

Add-Type -AssemblyName System.Windows.Forms, System.Drawing
[System.Windows.Forms.Application]::EnableVisualStyles()

# ---- single instance ----
$createdNew = $false
$mutex = New-Object System.Threading.Mutex($true, 'Local\ProxyTray_HomeReality', [ref]$createdNew)
if (-not $createdNew) { exit }

# ---- custom controls (smooth, anti-aliased, animated) ----
if (-not ('PopupForm' -as [type])) {
Add-Type -ReferencedAssemblies System.Windows.Forms, System.Drawing -TypeDefinition @'
using System;
using System.Drawing;
using System.Drawing.Drawing2D;
using System.Drawing.Text;
using System.Windows.Forms;

public static class Gfx {
    public static GraphicsPath RR(Rectangle r, int rad) {
        if (rad < 1) rad = 1;
        int d = rad * 2;
        GraphicsPath p = new GraphicsPath();
        p.AddArc(r.X, r.Y, d, d, 180, 90);
        p.AddArc(r.Right - d, r.Y, d, d, 270, 90);
        p.AddArc(r.Right - d, r.Bottom - d, d, d, 0, 90);
        p.AddArc(r.X, r.Bottom - d, d, d, 90, 90);
        p.CloseFigure();
        return p;
    }
    public static Color Lerp(Color a, Color b, float t) {
        if (t < 0) t = 0; if (t > 1) t = 1;
        return Color.FromArgb((int)(a.R + (b.R - a.R) * t), (int)(a.G + (b.G - a.G) * t), (int)(a.B + (b.B - a.B) * t));
    }
}

public class PopupForm : Form {
    public event EventHandler EndSession;
    public int CornerRadius = 10;
    public Color BorderColor = Color.Gray;
    Timer t = new Timer();
    double p; int dir; Point home; int slide;

    public PopupForm() { DoubleBuffered = true; t.Interval = 15; t.Tick += OnTick; }

    protected override CreateParams CreateParams {
        get { CreateParams cp = base.CreateParams; cp.ClassStyle |= 0x20000; cp.ExStyle |= 0x80; return cp; }
    }
    protected override void WndProc(ref Message m) {
        if (m.Msg == 0x16 && m.WParam != IntPtr.Zero) { EventHandler h = EndSession; if (h != null) h(this, EventArgs.Empty); }
        base.WndProc(ref m);
    }
    public void ShowAnimated(Point loc, int dy) {
        home = loc; slide = dy; p = 0; dir = 1;
        Opacity = 0;
        Location = new Point(loc.X, loc.Y + dy);
        Show(); Activate();
        t.Start();
    }
    public void HideAnimated() { if (!Visible) return; dir = -1; t.Start(); }
    public void HideNow() { t.Stop(); dir = 0; Hide(); }
    void OnTick(object s, EventArgs e) {
        p += dir * 0.14;
        bool hide = false;
        if (p >= 1) { p = 1; t.Stop(); }
        if (p <= 0) { p = 0; t.Stop(); hide = true; }
        double ease = 1 - Math.Pow(1 - p, 3);
        Opacity = ease;
        Location = new Point(home.X, home.Y + (int)((1 - ease) * slide));
        if (hide) Hide();
    }
    protected override void OnPaint(PaintEventArgs e) {
        base.OnPaint(e);
        e.Graphics.SmoothingMode = SmoothingMode.AntiAlias;
        using (Pen pen = new Pen(BorderColor, 1f))
        using (GraphicsPath path = Gfx.RR(new Rectangle(0, 0, Width - 1, Height - 1), CornerRadius))
            e.Graphics.DrawPath(pen, path);
    }
}

public class RoundPanel : Panel {
    public int Radius = 10;
    public Color Fill = Color.Gray;
    public RoundPanel() { DoubleBuffered = true; }
    protected override void OnPaint(PaintEventArgs e) {
        base.OnPaint(e);
        e.Graphics.SmoothingMode = SmoothingMode.AntiAlias;
        using (GraphicsPath path = Gfx.RR(new Rectangle(0, 0, Width - 1, Height - 1), Radius))
        using (SolidBrush b = new SolidBrush(Fill))
            e.Graphics.FillPath(b, path);
    }
}

public class RoundButton : Control {
    public Color Fill = Color.Gray, HoverFill = Color.DimGray;
    public int Radius = 8;
    float h; bool over, down;
    Timer t = new Timer();
    public RoundButton() {
        SetStyle(ControlStyles.AllPaintingInWmPaint | ControlStyles.OptimizedDoubleBuffer | ControlStyles.UserPaint | ControlStyles.ResizeRedraw, true);
        SetStyle(ControlStyles.Selectable, false);
        Cursor = Cursors.Hand;
        t.Interval = 15;
        t.Tick += delegate {
            float target = (over || down) ? 1f : 0f;
            h += (target - h) * 0.3f;
            if (Math.Abs(target - h) < 0.01f) { h = target; t.Stop(); }
            Invalidate();
        };
    }
    protected override void OnMouseEnter(EventArgs e) { over = true; t.Start(); base.OnMouseEnter(e); }
    protected override void OnMouseLeave(EventArgs e) { over = false; down = false; t.Start(); base.OnMouseLeave(e); }
    protected override void OnMouseDown(MouseEventArgs e) { down = true; t.Start(); base.OnMouseDown(e); }
    protected override void OnMouseUp(MouseEventArgs e) { down = false; t.Start(); base.OnMouseUp(e); }
    protected override void OnPaint(PaintEventArgs e) {
        Graphics g = e.Graphics;
        g.SmoothingMode = SmoothingMode.AntiAlias;
        using (SolidBrush bg = new SolidBrush(BackColor)) g.FillRectangle(bg, ClientRectangle);
        Color c = Gfx.Lerp(Fill, HoverFill, h);
        using (GraphicsPath path = Gfx.RR(new Rectangle(0, 0, Width - 1, Height - 1), Radius))
        using (SolidBrush b = new SolidBrush(c))
            g.FillPath(b, path);
        g.TextRenderingHint = TextRenderingHint.ClearTypeGridFit;
        using (StringFormat sf = new StringFormat())
        using (SolidBrush tb = new SolidBrush(ForeColor)) {
            sf.Alignment = StringAlignment.Center; sf.LineAlignment = StringAlignment.Center;
            g.DrawString(Text, Font, tb, new RectangleF(0, 0, Width, Height), sf);
        }
    }
}

public class ToggleSwitch : Control {
    public Color OffColor = Color.Gray, OnColor = Color.Green, BusyColor = Color.Orange;
    int state; float knob, pulse;
    Timer t = new Timer();
    public int State {
        get { return state; }
        set { if (state != value) { state = value; t.Start(); } }
    }
    public ToggleSwitch() {
        SetStyle(ControlStyles.AllPaintingInWmPaint | ControlStyles.OptimizedDoubleBuffer | ControlStyles.UserPaint | ControlStyles.ResizeRedraw, true);
        SetStyle(ControlStyles.Selectable, false);
        Cursor = Cursors.Hand;
        t.Interval = 15;
        t.Tick += delegate {
            float target = state == 1 ? 1f : (state == 2 ? 0.5f : 0f);
            knob += (target - knob) * 0.25f;
            if (Math.Abs(target - knob) < 0.005f) knob = target;
            if (state == 2) pulse += 0.14f;
            bool done = (knob == target) && state != 2;
            Invalidate();
            if (done) t.Stop();
        };
    }
    protected override void OnPaint(PaintEventArgs e) {
        Graphics g = e.Graphics;
        g.SmoothingMode = SmoothingMode.AntiAlias;
        using (SolidBrush bg = new SolidBrush(BackColor)) g.FillRectangle(bg, ClientRectangle);
        Color c = state == 2
            ? Gfx.Lerp(OffColor, BusyColor, 0.6f + 0.4f * (float)Math.Sin(pulse))
            : Gfx.Lerp(OffColor, OnColor, knob);
        int w = Width - 1, hh = Height - 1;
        using (GraphicsPath path = Gfx.RR(new Rectangle(0, 0, w, hh), hh / 2))
        using (SolidBrush b = new SolidBrush(c))
            g.FillPath(b, path);
        int pad = Math.Max(3, hh / 10);
        int d = hh - 2 * pad;
        float x = pad + (w - d - 2 * pad) * knob;
        using (SolidBrush sh = new SolidBrush(Color.FromArgb(55, 0, 0, 0)))
            g.FillEllipse(sh, x, pad + 2, d, d);
        g.FillEllipse(Brushes.White, x, pad, d, d);
    }
}
'@
}

# ---- settings (same as proxy_toggle.bat) ----
$ProxyName  = 'Home-Reality'
$ProxyHost  = 'play.rowstonsoftware.com'
$ProxyPort  = 443
$ProxyUUID  = 'a750d63d-3348-4f8b-be6a-be17606aff60'
$ProxyFlow  = 'xtls-rprx-vision'
$ProxySNI   = 'dl.google.com'
$ProxyFP    = 'chrome'
$ProxyPBK   = 'npglY0nkJZ16gJ-AmWkt30hf1zgikjNRN9BhPUOKVnQ'
$ProxySID   = '0123456789abcdef'
$LocalPort  = 2080
$LocalProxy = "127.0.0.1:$LocalPort"
$Dir        = $PSScriptRoot
$SingboxJson = Join-Path $Dir 'sing-box-config.json'
$XrayJson    = Join-Path $Dir 'xray-config.json'
$CoreLog     = Join-Path $Dir 'proxy_core.log'
$RegKey      = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings'
$RunKey      = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Run'
$Uri = "vless://$ProxyUUID@${ProxyHost}:$ProxyPort?encryption=none&flow=$ProxyFlow&security=reality&sni=$ProxySNI&fp=$ProxyFP&pbk=$ProxyPBK&sid=$ProxySID&type=tcp#$ProxyName"
$CoreProcs = 'sing-box','nekobox_core','nekoray_core','xray'

# ---- DPI scale ----
$gScr = [System.Drawing.Graphics]::FromHwnd([IntPtr]::Zero)
$script:scale = $gScr.DpiX / 96.0
$gScr.Dispose()
function S($n) { return [int][math]::Round($n * $script:scale) }

# ---- colors ----
function Hex($h) { return [System.Drawing.ColorTranslator]::FromHtml($h) }
$cBg    = Hex '#16181D'
$cCard  = Hex '#222630'
$cText  = Hex '#E8EAED'
$cMuted = Hex '#8A909C'
$cGreen = Hex '#34C759'
$cRed   = Hex '#FF453A'
$cAmber = Hex '#FF9F0A'
$cOff   = Hex '#3A3F4B'
$cBorder = Hex '#2C313C'

$script:busy = $false
$script:lastHide = 0

# ---- helpers ----
function New-LogoBitmap([bool]$on) {
    $bmp = New-Object System.Drawing.Bitmap 64, 64
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.SmoothingMode = 'AntiAlias'
    $g.Clear([System.Drawing.Color]::Transparent)

    $shield = New-Object System.Drawing.Drawing2D.GraphicsPath
    $shield.AddLine(32, 3, 56, 11)
    $shield.AddLine(56, 11, 56, 30)
    $shield.AddBezier(56, 30, 56, 46, 45, 57, 32, 61)
    $shield.AddBezier(32, 61, 19, 57, 8, 46, 8, 30)
    $shield.AddLine(8, 30, 8, 11)
    $shield.CloseFigure()

    if ($on) { $c1 = Hex '#3DF2A0'; $c2 = Hex '#0090E0'; $boltColor = [System.Drawing.Color]::White }
    else     { $c1 = Hex '#6E7585'; $c2 = Hex '#383D49'; $boltColor = Hex '#B4BAC6' }
    $rect = New-Object System.Drawing.RectangleF 0, 0, 64, 64
    $grad = New-Object System.Drawing.Drawing2D.LinearGradientBrush $rect, $c1, $c2, 65.0
    $g.FillPath($grad, $shield)

    # soft top highlight
    $g.SetClip($shield)
    $hi = New-Object System.Drawing.SolidBrush ([System.Drawing.Color]::FromArgb(48, 255, 255, 255))
    $g.FillEllipse($hi, -10, -26, 84, 56)
    $g.ResetClip()

    # lightning bolt
    $bolt = [System.Drawing.PointF[]]@(
        (New-Object System.Drawing.PointF 37, 10), (New-Object System.Drawing.PointF 20, 35),
        (New-Object System.Drawing.PointF 31, 35), (New-Object System.Drawing.PointF 27, 54),
        (New-Object System.Drawing.PointF 45, 26), (New-Object System.Drawing.PointF 34, 26),
        (New-Object System.Drawing.PointF 40, 10))
    $bb = New-Object System.Drawing.SolidBrush $boltColor
    $g.FillPolygon($bb, $bolt)
    $g.Dispose()
    return $bmp
}

function New-TrayIcon($bmp) { return [System.Drawing.Icon]::FromHandle($bmp.GetHicon()) }

function Find-Core {
    foreach ($c in @(
        @('sing-box.exe','singbox'), @('nekobox_core.exe','singbox'),
        @('xray.exe','xray'), @('nekoray_core.exe','xray'))) {
        $p = Join-Path $Dir $c[0]
        if (Test-Path $p) { return @{ Path = $p; Type = $c[1]; Name = $c[0] } }
    }
    return $null
}

function Get-CoreProc { return (Get-Process -Name $CoreProcs -ErrorAction SilentlyContinue | Select-Object -First 1) }
function Test-Running { return [bool](Get-CoreProc) }

function Stop-Cores {
    Get-Process -Name ($CoreProcs + 'nekoray','nekobox') -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
}

function Test-OurProxySet {
    $p = Get-ItemProperty $RegKey -ErrorAction SilentlyContinue
    return ($p.ProxyEnable -eq 1 -and $p.ProxyServer -eq $LocalProxy)
}

function Set-SysProxy([bool]$on) {
    if ($on) {
        Set-ItemProperty $RegKey -Name ProxyServer -Value $LocalProxy -Type String
        Set-ItemProperty $RegKey -Name ProxyEnable -Value 1 -Type DWord
    } else {
        Set-ItemProperty $RegKey -Name ProxyEnable -Value 0 -Type DWord
        Remove-ItemProperty $RegKey -Name ProxyServer -ErrorAction SilentlyContinue
    }
    [Native.Api]::InternetSetOption([IntPtr]::Zero, 39, [IntPtr]::Zero, 0) | Out-Null
    [Native.Api]::InternetSetOption([IntPtr]::Zero, 37, [IntPtr]::Zero, 0) | Out-Null
}

function Wait-Running([int]$ms) {
    $t = [Environment]::TickCount
    while (([Environment]::TickCount - $t) -lt $ms) {
        [System.Windows.Forms.Application]::DoEvents()
        Start-Sleep -Milliseconds 15
    }
    return (Test-Running)
}

function Start-Core {
    $c = Find-Core
    if (-not $c) { return 'No core found. Put sing-box.exe next to this script.' }
    Remove-Item $CoreLog -ErrorAction SilentlyContinue
    if ($c.Type -eq 'singbox') {
        if (-not (Test-Path $SingboxJson)) { return 'Missing sing-box-config.json' }
        $forms = @("run -c `"$SingboxJson`"")
    } else {
        if (-not (Test-Path $XrayJson)) { return 'Missing xray-config.json' }
        $forms = @("run -c `"$XrayJson`"", "-config `"$XrayJson`"", "run -config `"$XrayJson`"")
    }
    foreach ($f in $forms) {
        $cmd = "/c `"`"$($c.Path)`" $f > `"$CoreLog`" 2>&1`""
        Start-Process -FilePath "$env:SystemRoot\System32\cmd.exe" -ArgumentList $cmd -WindowStyle Hidden -WorkingDirectory $Dir
        if (Wait-Running 2500) { return $null }
    }
    $tail = ''
    if (Test-Path $CoreLog) { $tail = (Get-Content $CoreLog -Tail 4 -ErrorAction SilentlyContinue | Out-String).Trim() }
    if (-not $tail) { $tail = 'Core did not stay running.' }
    return $tail
}

# ---- first-run download of sing-box (get-singbox.ps1) ----
$GetScript = Join-Path $Dir 'get-singbox.ps1'
$DlLog     = Join-Path $Dir 'get-singbox.log'
$script:dl = $null
$script:dlHandled = $true

function Test-Downloading { return [bool]($script:dl -and -not $script:dl.HasExited) }

function Start-CoreDownload {
    if (Test-Downloading) { return }
    Remove-Item $DlLog -ErrorAction SilentlyContinue
    if (-not (Test-Path $GetScript)) { return }
    $script:dlHandled = $false
    $script:dl = Start-Process -FilePath "$env:SystemRoot\System32\WindowsPowerShell\v1.0\powershell.exe" `
        -ArgumentList "-NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File `"$GetScript`" -Quiet" `
        -WindowStyle Hidden -PassThru
}

function Get-DownloadError {
    $t = ''
    if (Test-Path $DlLog) { $t = (Get-Content $DlLog -Tail 2 -ErrorAction SilentlyContinue | Out-String).Trim() }
    if (-not $t) { $t = 'Check your internet connection, or put sing-box.exe next to this script.' }
    return "Couldn't download sing-box. $t"
}

# returns $null when a core is available, otherwise an error message
function Ensure-Core {
    if (Find-Core) { return $null }
    Start-CoreDownload
    while (Test-Downloading) {
        [System.Windows.Forms.Application]::DoEvents()
        Start-Sleep -Milliseconds 50
    }
    $script:dlHandled = $true
    if (Find-Core) { return $null }
    return (Get-DownloadError)
}

function Enable-Proxy {
    $script:busy = $true; $lblErr.Text = ''; Update-UI
    $touched = $false
    $err = Ensure-Core
    if (-not $err) {
        $touched = $true
        Stop-Cores
        Set-SysProxy $true
        $err = Start-Core
    }
    if ($err) {
        if ($touched) { Set-SysProxy $false; Stop-Cores }
        $lblErr.Text = if ($err.Length -gt 220) { $err.Substring(0,220) + '...' } else { $err }
        $tray.ShowBalloonTip(3000, $ProxyName, 'Failed to start. Open the panel for details.', 'Error')
    }
    $script:busy = $false; Update-UI
}

function Disable-Proxy {
    $script:busy = $true; $lblErr.Text = ''; Update-UI
    Stop-Cores
    Set-SysProxy $false
    $null = Wait-Running 250
    $script:busy = $false; Update-UI
}

function Toggle-Proxy {
    if ($script:busy) { return }
    if (Test-Running) { Disable-Proxy } else { Enable-Proxy }
}

function Format-Up($ts) {
    if ($ts.TotalHours -ge 1) { return ('{0}h {1}m' -f [int][math]::Floor($ts.TotalHours), $ts.Minutes) }
    if ($ts.TotalMinutes -ge 1) { return ('{0}m' -f [int][math]::Floor($ts.TotalMinutes)) }
    return 'just now'
}

# ---- tray icons ----
$bmpOn  = New-LogoBitmap $true
$bmpOff = New-LogoBitmap $false
$iconOn  = New-TrayIcon $bmpOn
$iconOff = New-TrayIcon $bmpOff

# ---- popup form ----
$form = New-Object PopupForm
$form.FormBorderStyle = 'None'
$form.StartPosition = 'Manual'
$form.ShowInTaskbar = $false
$form.TopMost = $true
$form.ClientSize = New-Object System.Drawing.Size((S 320), (S 448))
$form.BackColor = $cBg
$form.ForeColor = $cText
$form.Font = New-Object System.Drawing.Font('Segoe UI', 9)
$form.BorderColor = $cBorder
$null = $form.Handle

# smooth native rounded corners on Windows 11; fallback region on Windows 10
$pref = 2
$dwm = [Native.Api]::DwmSetWindowAttribute($form.Handle, 33, [ref]$pref, 4)
if ($dwm -eq 0) {
    $form.CornerRadius = S 8
} else {
    $form.CornerRadius = S 12
    $path = New-Object System.Drawing.Drawing2D.GraphicsPath
    $d = (S 12) * 2; $W = $form.Width; $H = $form.Height
    $path.AddArc(0, 0, $d, $d, 180, 90); $path.AddArc($W - $d, 0, $d, $d, 270, 90)
    $path.AddArc($W - $d, $H - $d, $d, $d, 0, 90); $path.AddArc(0, $H - $d, $d, $d, 90, 90)
    $path.CloseFigure()
    $form.Region = New-Object System.Drawing.Region $path
}

$aL = [System.Drawing.ContentAlignment]::MiddleLeft
$aC = [System.Drawing.ContentAlignment]::MiddleCenter
$fReg = [System.Drawing.FontStyle]::Regular
$fBold = [System.Drawing.FontStyle]::Bold

function New-Label($parent, $text, $x, $y, $w, $h, $size, $style, $color, $align) {
    $l = New-Object System.Windows.Forms.Label
    $l.Text = $text
    $l.Location = New-Object System.Drawing.Point((S $x), (S $y))
    $l.Size = New-Object System.Drawing.Size((S $w), (S $h))
    $l.Font = New-Object System.Drawing.Font('Segoe UI', $size, $style)
    $l.ForeColor = $color; $l.BackColor = [System.Drawing.Color]::Transparent
    $l.TextAlign = $align
    $parent.Controls.Add($l)
    return $l
}

$logo = New-Object System.Windows.Forms.PictureBox
$logo.Location = New-Object System.Drawing.Point((S 20), (S 16))
$logo.Size = New-Object System.Drawing.Size((S 40), (S 40))
$logo.SizeMode = 'Zoom'
$logo.BackColor = [System.Drawing.Color]::Transparent
$logo.Image = $bmpOff
$form.Controls.Add($logo)
$null = New-Label $form $ProxyName 68 16 232 26 15 $fBold $cText $aL
$null = New-Label $form 'VLESS Reality' 68 40 232 20 9 $fReg $cMuted $aL

# switch
$sw = New-Object ToggleSwitch
$sw.Size = New-Object System.Drawing.Size((S 132), (S 66))
$sw.Location = New-Object System.Drawing.Point((S 94), (S 88))
$sw.BackColor = $cBg
$sw.OffColor = $cOff; $sw.OnColor = $cGreen; $sw.BusyColor = $cAmber
$sw.Add_Click({ Toggle-Proxy })
$form.Controls.Add($sw)

$lblStatus = New-Label $form 'Disconnected' 20 164 280 30 15 $fBold $cRed $aC
$lblSub    = New-Label $form '' 20 194 280 20 9 $fReg $cMuted $aC

# info card
$card = New-Object RoundPanel
$card.Location = New-Object System.Drawing.Point((S 20), (S 228))
$card.Size = New-Object System.Drawing.Size((S 280), (S 108))
$card.BackColor = $cBg
$card.Fill = $cCard
$card.Radius = S 10
$form.Controls.Add($card)

$tip = New-Object System.Windows.Forms.ToolTip
function Add-Row($label, $y) {
    $null = New-Label $card $label 14 $y 56 22 9 $fReg $cMuted $aL
    $v = New-Label $card '' 72 $y 196 22 9 $fReg $cText $aL
    $v.AutoEllipsis = $true
    return $v
}
$vHost = Add-Row 'Host'  9;   $vHost.Text = "${ProxyHost}:$ProxyPort"
$vLoc  = Add-Row 'Local' 33;  $vLoc.Text = $LocalProxy
$vCore = Add-Row 'Core'  57
$vType = Add-Row 'Type'  81

$lblErr = New-Label $form '' 20 344 280 40 8 $fReg $cRed $aL

# buttons
function New-Btn($text, $x, $onClick) {
    $b = New-Object RoundButton
    $b.Text = $text
    $b.Location = New-Object System.Drawing.Point((S $x), (S 394))
    $b.Size = New-Object System.Drawing.Size((S 64), (S 36))
    $b.BackColor = $cBg; $b.ForeColor = $cText
    $b.Fill = $cCard; $b.HoverFill = $cOff
    $b.Radius = S 8
    $b.Font = New-Object System.Drawing.Font('Segoe UI', 9)
    $b.Add_Click($onClick)
    $form.Controls.Add($b)
    return $b
}
$btnCopy = New-Btn 'Copy URI' 20 {
    [System.Windows.Forms.Clipboard]::SetText($Uri)
    $btnCopy.Text = 'Copied!'
    $t = New-Object System.Windows.Forms.Timer; $t.Interval = 1500
    $t.Add_Tick({ $btnCopy.Text = 'Copy URI'; $this.Stop(); $this.Dispose() }); $t.Start()
}
$null = New-Btn 'Log' 92 {
    if (Test-Path $CoreLog) { Start-Process notepad.exe $CoreLog } else { $lblErr.Text = 'No log yet.' }
}
$null = New-Btn 'Folder' 164 { Start-Process explorer.exe $Dir }
$null = New-Btn 'Quit' 236 { Quit-App }

$form.Add_Deactivate({ $form.HideAnimated(); $script:lastHide = [Environment]::TickCount })

# turn everything off cleanly when Windows shuts down / signs out
$form.add_EndSession({
    try { Stop-Cores; Set-SysProxy $false } catch {}
})

# ---- tray ----
$tray = New-Object System.Windows.Forms.NotifyIcon
$tray.Icon = $iconOff
$tray.Visible = $true

$menu = New-Object System.Windows.Forms.ContextMenuStrip
$miOpen   = $menu.Items.Add('Open panel')
$miToggle = $menu.Items.Add('Turn on')
$null = $menu.Items.Add('-')
$miAuto   = $menu.Items.Add('Start with Windows')
$miQuit   = $menu.Items.Add('Quit')
$tray.ContextMenuStrip = $menu

function Show-Popup {
    Update-UI
    $cur = [System.Windows.Forms.Cursor]::Position
    $scr = [System.Windows.Forms.Screen]::FromPoint($cur).WorkingArea
    $m = S 12
    if ($cur.X -lt ($scr.Left + $scr.Width / 2)) { $x = $scr.Left + $m } else { $x = $scr.Right - $form.Width - $m }
    if ($cur.Y -lt ($scr.Top + $scr.Height / 2)) { $y = $scr.Top + $m; $dy = -(S 14) } else { $y = $scr.Bottom - $form.Height - $m; $dy = S 14 }
    $form.ShowAnimated((New-Object System.Drawing.Point($x, $y)), $dy)
}

function Quit-App {
    $form.HideNow()
    if (Test-Running) {
        $r = [System.Windows.Forms.MessageBox]::Show('Turn off the proxy before quitting?', $ProxyName, 'YesNoCancel', 'Question')
        if ($r -eq 'Cancel') { return }
        if ($r -eq 'Yes') { Stop-Cores; Set-SysProxy $false }
    }
    $tray.Visible = $false
    $tray.Dispose()
    [System.Windows.Forms.Application]::Exit()
}

$tray.Add_MouseUp({
    param($s, $e)
    if ($e.Button -eq 'Left' -and ([Environment]::TickCount - $script:lastHide) -gt 250) { Show-Popup }
})
$miOpen.Add_Click({ Show-Popup })
$miToggle.Add_Click({ Toggle-Proxy })
$miQuit.Add_Click({ Quit-App })
$miAuto.Add_Click({
    $launcher = Join-Path $Dir 'ProxyTray.vbs'
    if (Get-ItemProperty $RunKey -Name 'ProxyTray' -ErrorAction SilentlyContinue) {
        Remove-ItemProperty $RunKey -Name 'ProxyTray' -ErrorAction SilentlyContinue
    } else {
        Set-ItemProperty $RunKey -Name 'ProxyTray' -Value "wscript.exe `"$launcher`"" -Type String
    }
})
$menu.Add_Opening({
    $miToggle.Text = if (Test-Running) { 'Turn off' } else { 'Turn on' }
    $miAuto.Checked = [bool](Get-ItemProperty $RunKey -Name 'ProxyTray' -ErrorAction SilentlyContinue)
})

function Update-UI {
    $proc = Get-CoreProc
    $running = [bool]$proc
    $dling = Test-Downloading
    $c = Find-Core
    if ($c) {
        $vCore.Text = $c.Name; $vCore.ForeColor = $cText; $tip.SetToolTip($vCore, $c.Path)
        $vType.Text = $c.Type
    } elseif ($dling) {
        $vCore.Text = 'Downloading...'; $vCore.ForeColor = $cAmber; $vType.Text = '-'
    } else {
        $vCore.Text = 'NOT FOUND'; $vCore.ForeColor = $cRed; $vType.Text = '-'
    }
    if ($script:busy -or $dling) {
        $sw.State = 2
        $lblStatus.Text = 'Working...'; $lblStatus.ForeColor = $cAmber
        $lblSub.Text = if ($dling) { 'Downloading sing-box (first run)...' } else { '' }
    } elseif ($running) {
        $sw.State = 1
        $lblStatus.Text = 'Connected'; $lblStatus.ForeColor = $cGreen
        $up = ''
        try { $up = 'Up ' + (Format-Up ((Get-Date) - $proc.StartTime)) + '   ' + [char]0x00B7 + '   ' } catch {}
        $lblSub.Text = $up + $LocalProxy
        $tray.Icon = $iconOn; $tray.Text = "$ProxyName - Connected"; $logo.Image = $bmpOn
    } else {
        $sw.State = 0
        $lblStatus.Text = 'Disconnected'; $lblStatus.ForeColor = $cRed; $lblSub.Text = 'Tap the switch to connect'
        $tray.Icon = $iconOff; $tray.Text = "$ProxyName - Disconnected"; $logo.Image = $bmpOff
    }
}

# keep the system proxy honest: if the core died/was killed, don't leave Windows pointing at a dead port
function Clear-Stale {
    if (-not $script:busy -and -not (Test-Running) -and (Test-OurProxySet)) { Set-SysProxy $false }
}

$poll = New-Object System.Windows.Forms.Timer
$poll.Interval = 2000
$poll.Add_Tick({
    if (-not $script:busy) {
        if ($script:dl -and -not $script:dlHandled -and $script:dl.HasExited) {
            $script:dlHandled = $true
            if (Find-Core) { $tray.ShowBalloonTip(3000, $ProxyName, 'sing-box downloaded. Ready to turn on.', 'Info') }
            else { $lblErr.Text = (Get-DownloadError); $tray.ShowBalloonTip(3000, $ProxyName, "Couldn't download sing-box. Open the panel for details.", 'Error') }
        }
        Clear-Stale; Update-UI
    }
})
$poll.Start()

# Open automatically when Windows starts (only opens the tray app - it never turns the proxy on by itself).
# Turned on once on first run; if you switch it off from the right-click menu it stays off.
try {
    $runVal  = "wscript.exe `"$(Join-Path $Dir 'ProxyTray.vbs')`""
    $initKey = 'HKCU:\Software\ProxyTray'
    if (-not (Test-Path $initKey)) { New-Item $initKey -Force | Out-Null }
    $cur = (Get-ItemProperty $RunKey -Name 'ProxyTray' -ErrorAction SilentlyContinue).ProxyTray
    if (-not (Get-ItemProperty $initKey -Name 'AutoStartInit' -ErrorAction SilentlyContinue)) {
        Set-ItemProperty $RunKey -Name 'ProxyTray' -Value $runVal -Type String
        Set-ItemProperty $initKey -Name 'AutoStartInit' -Value 1 -Type DWord
    } elseif ($cur -and $cur -ne $runVal) {
        Set-ItemProperty $RunKey -Name 'ProxyTray' -Value $runVal -Type String   # folder was moved
    }
} catch {}

if (-not (Find-Core)) { Start-CoreDownload }
Clear-Stale
Update-UI
[System.Windows.Forms.Application]::Run((New-Object System.Windows.Forms.ApplicationContext))
$mutex.ReleaseMutex()
