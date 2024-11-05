use actix_cloud::response_build::generate_response;

fn main() {
    generate_response("", "response", "response.rs").unwrap();
}
