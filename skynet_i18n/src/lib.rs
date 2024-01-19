use quote::quote;
use rust_i18n_support::{is_debug, load_locales};
use std::{collections::HashMap, env, path};
use syn::parse;

#[derive(Debug)]
struct Option {
    locales_path: String,
}

impl parse::Parse for Option {
    fn parse(input: parse::ParseStream) -> parse::Result<Self> {
        let locales_path = input.parse::<syn::LitStr>()?.value();

        Ok(Self { locales_path })
    }
}

/// Init I18n translations.
///
/// This will load all translations by glob `**/*.yml` from the given path.
///
/// ```ignore
/// i18n!("locales");
/// ```
///
/// # Panics
///
/// Panics is variable `CARGO_MANIFEST_DIR` is empty.
#[proc_macro]
pub fn i18n(input: proc_macro::TokenStream) -> proc_macro::TokenStream {
    let option = match syn::parse::<Option>(input) {
        Ok(input) => input,
        Err(err) => return err.to_compile_error().into(),
    };

    // CARGO_MANIFEST_DIR is current build directory
    let cargo_dir = env::var("CARGO_MANIFEST_DIR").expect("CARGO_MANIFEST_DIR is empty");
    let current_dir = path::PathBuf::from(cargo_dir);
    let locales_path = current_dir.join(option.locales_path);

    let data = load_locales(&locales_path.display().to_string(), |_| false);
    let mut translation = HashMap::new();
    for (lang, mp) in data {
        for (k, v) in mp {
            translation.insert(format!("{lang}.{k}"), v);
        }
    }
    let code = generate_code(&translation);

    if is_debug() {
        println!("{code}");
    }

    code.into()
}

fn generate_code(data: &HashMap<String, String>) -> proc_macro2::TokenStream {
    let mut locales = Vec::<proc_macro2::TokenStream>::new();

    for (k, v) in data {
        let k = k.to_owned();
        let v = v.to_owned();

        locales.push(quote! {
            #k => #v,
        });
    }

    // result
    quote! {
        ::skynet::map! [
            #(#locales)*
            "" => ""    // eat last comma
        ]
    }
}
