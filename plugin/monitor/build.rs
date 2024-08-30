use actix_cloud::response::generate_response;

fn main() {
    generate_response("skynet_api::", "response", "response.rs").unwrap();
}
