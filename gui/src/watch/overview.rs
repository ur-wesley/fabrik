use std::path::Path;
use std::sync::mpsc::Sender;

use crate::fabrik::fetch_graph_overview;
use crate::model::Action;

/// Instant overview fetch (`graph overview --json`): degree-ordered symbols
/// for the empty Lookup canvas. Answers with `LookupOverviewLoaded` so the
/// state caches it per repo; failures answer with `LookupOverviewFailed` so
/// the state keeps a persistent pane card (board state stays untouched).
pub(super) fn dispatch_lookup_overview(action_tx: &Sender<Action>, repo: &Path, budget: u32) {
    match fetch_graph_overview(repo, budget) {
        Ok(result) => {
            let _ = action_tx.send(Action::LookupOverviewLoaded {
                repo: repo.to_path_buf(),
                result,
            });
        }
        Err(err) => {
            let _ = action_tx.send(Action::LookupOverviewFailed {
                repo: repo.to_path_buf(),
                message: err.to_string(),
            });
        }
    }
}
