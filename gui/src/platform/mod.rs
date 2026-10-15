/// True when `--glass-debug` is on the command line or `NEON_GLASS_DEBUG=1` is
/// set. Enables verbose DWM logging and a
/// tinted debug overlay in `window_chrome::shell`.
#[must_use]
pub fn glass_debug_enabled() -> bool {
    std::env::args().any(|arg| arg == "--glass-debug")
        || std::env::var("NEON_GLASS_DEBUG").as_deref() == Ok("1")
}

#[cfg(target_os = "windows")]
mod windows;

#[cfg(target_os = "windows")]
pub use windows::refresh_glass;

#[cfg(target_os = "macos")]
mod macos;

#[cfg(target_os = "macos")]
pub use macos::refresh_glass;

#[cfg(not(any(target_os = "windows", target_os = "macos")))]
pub fn refresh_glass(_frame: Option<&eframe::Frame>, _focused: bool) -> bool {
    false
}

#[cfg(test)]
mod tests {
    use super::glass_debug_enabled;

    #[test]
    fn glass_diagnostics_default_quiet_with_env_opt_in() {
        // Merged into a single test: both cases touch the process-wide
        // `NEON_GLASS_DEBUG` env var, so running them as two `#[test]` fns
        // races under parallel `cargo test` (one removes while the other
        // sets). Sequential steps here keep default `cargo test` green.
        let key = "NEON_GLASS_DEBUG";
        let saved = std::env::var(key).ok();
        // Restore the pre-test value on drop so a mid-test panic cannot
        // leak env state into other tests.
        struct Restore {
            key: &'static str,
            saved: Option<String>,
        }
        impl Drop for Restore {
            fn drop(&mut self) {
                // SAFETY: single-threaded within this test; restores the
                // pre-test value only.
                match &self.saved {
                    Some(value) => unsafe { std::env::set_var(self.key, value) },
                    None => unsafe { std::env::remove_var(self.key) },
                }
            }
        }
        let _restore = Restore { key, saved };
        // SAFETY: no other test in this crate touches this env var; all
        // access is sequenced inside this one test.
        unsafe { std::env::remove_var(key) };
        assert!(
            !glass_debug_enabled(),
            "glass diagnostics must be quiet by default (no --glass-debug / NEON_GLASS_DEBUG=1)"
        );
        // SAFETY: see above.
        unsafe { std::env::set_var(key, "1") };
        assert!(
            glass_debug_enabled(),
            "NEON_GLASS_DEBUG=1 must enable glass diagnostics"
        );
    }
}
