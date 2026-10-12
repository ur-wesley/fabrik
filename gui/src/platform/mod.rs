/// True when `--glass-debug` is on the command line or `NEON_GLASS_DEBUG=1` is
/// set. Enables verbose DWM logging and a
/// tinted debug overlay in `window_chrome::shell`.
#[must_use]
pub fn glass_debug_enabled() -> bool {
    std::env::args().any(|arg| arg == "--glass-debug")
        || std::env::var("NEON_GLASS_DEBUG").as_deref() == Ok("1")
}

#[cfg(windows)]
pub use windows::*;

#[cfg(target_os = "macos")]
mod macos;

#[cfg(target_os = "macos")]
pub use macos::refresh_glass;

#[cfg(not(any(windows, target_os = "macos")))]
pub fn refresh_glass(_frame: Option<&eframe::Frame>, _focused: bool) -> bool {
    false
}
