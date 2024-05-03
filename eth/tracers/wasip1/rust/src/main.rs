static mut GLOBAL: i32 = 0;

fn main() {
    println!("Hello, world!");
    unsafe {
        GLOBAL = 2;
    }
}

#[no_mangle]
pub extern "C" fn onBlockchainInit() {
    println!("Hello, world blockchain! {}", unsafe { GLOBAL });
}
