#[cfg(target_os = "macos")]
static MACOS_SURFACE_READY: std::sync::atomic::AtomicBool =
    std::sync::atomic::AtomicBool::new(false);

#[cfg(target_os = "macos")]
mod macos_cgs {
    use objc2_foundation::NSInteger;
    use std::ffi::c_void;

    #[link(name = "ApplicationServices", kind = "framework")]
    unsafe extern "C" {
        fn CGSMainConnectionID() -> *mut c_void;
        fn CGSSetWindowBackgroundBlurRadius(
            connection_id: *mut c_void,
            window_id: NSInteger,
            radius: i64,
        ) -> i32;
    }

    pub fn set_window_background_blur(window_id: NSInteger, radius: i64) {
        if radius <= 0 {
            return;
        }
        unsafe {
            let connection = CGSMainConnectionID();
            if connection.is_null() {
                return;
            }
            CGSSetWindowBackgroundBlurRadius(connection, window_id, radius);
        }
    }
}

/// `window-vibrancy` can cover the wgpu `CAMetalLayer`; we clear it and use
/// CGS window blur (same private API winit uses) plus shell tints for frost.
#[cfg(target_os = "macos")]
fn configure_macos_transparent_surface(frame: &eframe::Frame) -> bool {
    use super::glass_debug_enabled;
    use objc2_app_kit::NSView;
    use raw_window_handle::{HasWindowHandle, RawWindowHandle};
    use raw_window_metal::Layer;

    let Ok(handle) = frame.window_handle() else {
        return false;
    };
    let RawWindowHandle::AppKit(appkit) = handle.as_raw() else {
        return false;
    };

    let _ = window_vibrancy::clear_vibrancy(frame);

    let metal_wrapper = unsafe { Layer::from_ns_view(appkit.ns_view) };
    let metal_ptr = metal_wrapper.as_ptr();
    // SAFETY: `raw-window-metal` documents `as_ptr` as a live `CAMetalLayer`.
    let metal = unsafe { metal_ptr.cast::<objc2_quartz_core::CAMetalLayer>().as_ref() };
    metal.setOpaque(false);

    let ns_view_ptr = appkit.ns_view.as_ptr();
    if ns_view_ptr.is_null() {
        return false;
    }
    // SAFETY: Handle comes from the live AppKit window.
    let Some(view) = (unsafe { ns_view_ptr.cast::<NSView>().as_ref() }) else {
        return false;
    };
    if let Some(window) = view.window() {
        macos_cgs::set_window_background_blur(window.windowNumber(), 80);
    }

    if glass_debug_enabled() {
        eprintln!("[glass] macOS metal opaque=false + CGS blur radius 80");
    }
    true
}

#[cfg(target_os = "macos")]
pub fn refresh_glass(frame: Option<&eframe::Frame>, _focused: bool) -> bool {
    if MACOS_SURFACE_READY.load(std::sync::atomic::Ordering::Relaxed) {
        return true;
    }
    let Some(frame) = frame else {
        return false;
    };
    if configure_macos_transparent_surface(frame) {
        MACOS_SURFACE_READY.store(true, std::sync::atomic::Ordering::Relaxed);
        return true;
    }
    false
}
