//! Which `fabrik` binary the GUI spawns.
//!
//! The GUI shells `FABRIK_BIN` (else `fabrik` from PATH) for every read and
//! write, always off the UI thread. On Windows the configured path resolves
//! deterministically (see [`resolve_fabrik_bin`); everywhere else the
//! configured value is used as-is.

pub fn fabrik_bin() -> String {
    resolve_fabrik_bin(&std::env::var("FABRIK_BIN").unwrap_or_else(|_| "fabrik".to_owned()))
}

/// Resolve the configured CLI path to the file `Command` will execute.
///
/// On Windows, spawning an extensionless path runs its `.exe` sibling when
/// one exists (CreateProcess appends `.exe`), so a stale `fabrik.exe` next
/// to a fresh `fabrik` silently hijacks execution (cli-umi). Rules:
/// - exact-path match wins: when the configured path names an existing
///   file it is used as-is — the resolver never prefers a sibling over the
///   file the caller named, so a stale shadow cannot win at this layer;
/// - otherwise fall back to the sibling with/without `.exe` (either
///   direction), so an `FABRIK_BIN` written for the other platform layout
///   still finds the fresh build;
/// - bare names (`fabrik`) and missing files pass through unchanged, so
///   PATH (+PATHEXT) lookup and spawn errors behave exactly as before.
/// The returned path is canonicalized (absolute, symlinks resolved) so the
/// old-binary error card names the file that actually failed.
/// NOTE: when both the extensionless file and its `.exe` sibling exist,
/// only the returned path is deterministic — the OS still applies its own
/// `.exe` search to extensionless spawns, so leftover shadow pairs must be
/// deleted (both names are gitignored build artifacts). Everywhere else
/// this is the identity function.
fn resolve_fabrik_bin(configured: &str) -> String {
    #[cfg(windows)]
    {
        resolve_fabrik_bin_windows(configured)
    }
    #[cfg(not(windows))]
    {
        configured.to_owned()
    }
}

#[cfg(windows)]
fn resolve_fabrik_bin_windows(configured: &str) -> String {
    use std::path::Path;
    let path = Path::new(configured);
    // Bare `fabrik` / `fabrik.exe`: PATH (+PATHEXT) lookup, unchanged.
    if path.parent().is_none_or(|p| p.as_os_str().is_empty()) {
        return configured.to_owned();
    }
    if path.is_file() {
        return canonical_display(path);
    }
    if let Some(sibling) = sibling_with_or_without_exe(configured)
        && Path::new(&sibling).is_file()
    {
        return canonical_display(Path::new(&sibling));
    }
    canonical_display(path)
}

/// Sibling with `.exe` appended, or with a trailing `.exe` (any case)
/// stripped — the fallback the other platform layout needs.
#[cfg(windows)]
fn sibling_with_or_without_exe(configured: &str) -> Option<String> {
    if configured.len() >= 4 && configured[configured.len() - 4..].eq_ignore_ascii_case(".exe") {
        Some(configured[..configured.len() - 4].to_owned())
    } else {
        Some(format!("{configured}.exe"))
    }
}

#[cfg(windows)]
fn canonical_display(path: &std::path::Path) -> String {
    match std::fs::canonicalize(path) {
        Ok(abs) => strip_verbatim_prefix(&abs.to_string_lossy()),
        Err(_) => path.to_string_lossy().into_owned(),
    }
}

/// `canonicalize` returns verbatim `\\?\C:\…` paths; the shortened form
/// executes identically and reads better in error cards.
#[cfg(windows)]
fn strip_verbatim_prefix(canonical: &str) -> String {
    if let Some(rest) = canonical.strip_prefix(r"\\?\UNC\") {
        return format!(r"\\{rest}");
    }
    if let Some(rest) = canonical.strip_prefix(r"\\?\") {
        return rest.to_owned();
    }
    canonical.to_owned()
}

#[cfg(test)]
mod tests {
    use std::path::{Path, PathBuf};

    use super::resolve_fabrik_bin;

    fn unique_tmp_dir(tag: &str) -> PathBuf {
        let dir = std::env::temp_dir().join(format!("neon-{tag}-{}", std::process::id()));
        let _ = std::fs::remove_dir_all(&dir);
        std::fs::create_dir_all(&dir).expect("tmp dir");
        dir
    }

    fn touch(dir: &Path, name: &str) -> PathBuf {
        let file = dir.join(name);
        std::fs::write(&file, b"x").expect("tmp file");
        file
    }

    #[cfg(windows)]
    #[test]
    fn bare_names_pass_through_for_path_lookup() {
        // No directory part: PATH (+PATHEXT) lookup must behave as before.
        assert_eq!(resolve_fabrik_bin("fabrik"), "fabrik");
        assert_eq!(resolve_fabrik_bin("fabrik.exe"), "fabrik.exe");
    }

    #[cfg(windows)]
    #[test]
    fn exact_path_wins_over_exe_sibling() {
        // cli-umi: both exist (fresh extensionless + stale .exe). The
        // resolver deterministically returns the configured file and never
        // prefers the sibling on its own.
        let dir = unique_tmp_dir("bin-exact");
        touch(&dir, "fabrik");
        touch(&dir, "fabrik.exe");
        let configured = dir.join("fabrik").to_string_lossy().into_owned();
        let resolved = resolve_fabrik_bin(&configured);
        assert!(resolved.ends_with("fabrik"), "got {resolved:?}");
        assert!(!resolved.ends_with(".exe"), "got {resolved:?}");
        assert!(Path::new(&resolved).is_absolute(), "got {resolved:?}");
        assert!(!resolved.starts_with(r"\\?\"), "got {resolved:?}");
        let _ = std::fs::remove_dir_all(&dir);
    }

    #[cfg(windows)]
    #[test]
    fn missing_exact_falls_back_to_exe_sibling() {
        // Old FABRIK_BIN (extensionless) + new layout (only fabrik.exe).
        let dir = unique_tmp_dir("bin-fallback-exe");
        touch(&dir, "fabrik.exe");
        let configured = dir.join("fabrik").to_string_lossy().into_owned();
        let resolved = resolve_fabrik_bin(&configured);
        assert!(resolved.ends_with("fabrik.exe"), "got {resolved:?}");
        assert!(Path::new(&resolved).is_absolute(), "got {resolved:?}");
        let _ = std::fs::remove_dir_all(&dir);
    }

    #[cfg(windows)]
    #[test]
    fn missing_exe_configured_falls_back_to_extensionless_sibling() {
        // New FABRIK_BIN (.exe) + old layout (only extensionless file).
        let dir = unique_tmp_dir("bin-fallback-bare");
        touch(&dir, "fabrik");
        let configured = dir.join("fabrik.exe").to_string_lossy().into_owned();
        let resolved = resolve_fabrik_bin(&configured);
        assert!(resolved.ends_with("fabrik"), "got {resolved:?}");
        assert!(!resolved.ends_with(".exe"), "got {resolved:?}");
        let _ = std::fs::remove_dir_all(&dir);
    }

    #[cfg(windows)]
    #[test]
    fn missing_everything_passes_through_unchanged() {
        // Spawn error surfaces as before; no sibling invented.
        let dir = unique_tmp_dir("bin-missing");
        let configured = dir.join("fabrik").to_string_lossy().into_owned();
        assert_eq!(resolve_fabrik_bin(&configured), configured);
        let _ = std::fs::remove_dir_all(&dir);
    }

    #[cfg(not(windows))]
    #[test]
    fn resolution_is_identity_off_windows() {
        assert_eq!(resolve_fabrik_bin("fabrik"), "fabrik");
        assert_eq!(resolve_fabrik_bin("/tmp/cli/fabrik"), "/tmp/cli/fabrik");
    }
}
