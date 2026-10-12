#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub enum Screen {
    #[default]
    Board,
    Detail,
    Memory,
    Lookup,
    Settings,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub enum ViewMode {
    #[default]
    Kanban,
    List,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum Filter {
    Ready,
    Open,
    Wip,
    Blocked,
    Done,
}

pub const ALL_LANES: [Filter; 5] = [
    Filter::Ready,
    Filter::Open,
    Filter::Wip,
    Filter::Blocked,
    Filter::Done,
];

impl Filter {
    pub const fn key(self) -> &'static str {
        match self {
            Self::Ready => "ready",
            Self::Open => "open",
            Self::Wip => "wip",
            Self::Blocked => "blocked",
            Self::Done => "done",
        }
    }

    pub const fn label(self) -> &'static str {
        match self {
            Self::Ready => "Ready",
            Self::Open => "Open",
            Self::Wip => "WIP",
            Self::Blocked => "Blocked",
            Self::Done => "Done",
        }
    }

    pub fn from_key(key: &str) -> Option<Self> {
        match key {
            "ready" => Some(Self::Ready),
            "open" => Some(Self::Open),
            "wip" => Some(Self::Wip),
            "blocked" => Some(Self::Blocked),
            "done" => Some(Self::Done),
            _ => None,
        }
    }
}
