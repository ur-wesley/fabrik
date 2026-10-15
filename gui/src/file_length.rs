#[cfg(test)]
mod tests {
    use std::fs;
    use std::path::{Path, PathBuf};

    const MAX_LINES: usize = 500;

    #[test]
    fn src_rust_files_respect_line_cap() {
        let src = PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("src");
        let mut offenders = Vec::new();
        walk_rs(&src, &mut offenders);
        offenders.sort();
        assert!(
            offenders.is_empty(),
            "source files over {} lines:\n{}",
            MAX_LINES,
            offenders.join("\n")
        );
    }

    fn walk_rs(dir: &Path, offenders: &mut Vec<String>) {
        let entries = fs::read_dir(dir).expect("read src tree");
        for entry in entries.flatten() {
            let path = entry.path();
            if path.is_dir() {
                walk_rs(&path, offenders);
                continue;
            }
            if path.extension().is_some_and(|ext| ext == "rs") {
                let text = fs::read_to_string(&path).expect("read source file");
                let lines = text.lines().count();
                if lines > MAX_LINES {
                    let rel = path
                        .strip_prefix(env!("CARGO_MANIFEST_DIR"))
                        .unwrap_or(&path);
                    offenders.push(format!("{} ({} lines)", rel.display(), lines));
                }
            }
        }
    }
}
