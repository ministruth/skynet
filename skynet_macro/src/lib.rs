use proc_macro::TokenStream;
use quote::{format_ident, quote};
use syn::{
    parse_macro_input, parse_quote, ExprPath, FnArg, Ident, ImplItem, ItemFn, ItemImpl, ItemStruct,
    ItemTrait, Pat, TraitItem,
};

/// Implement common request param methods.
///
/// # Examples
/// ```ignore
/// #[common_req(Column)]
/// #[derive(Debug, Validate, Deserialize)]
/// pub struct GetReq {
///     #[serde(flatten)]
///     #[validate]
///     page: PaginationParam,
///     #[serde(flatten)]
///     #[validate]
///     time: TimeParam,
/// }
/// ```
#[proc_macro_attribute]
pub fn common_req(attr: TokenStream, input: TokenStream) -> TokenStream {
    let attr = parse_macro_input!(attr as ExprPath);
    let input = parse_macro_input!(input as ItemStruct);
    let name = &input.ident;

    quote! {
        #input
        impl #name {
            pub fn common_cond(&self) -> skynet_api::request::Condition {
                let mut cond = skynet_api::request::Condition::new(skynet_api::sea_orm::Condition::all()).add_page(self.page.clone());
                skynet_api::build_time_cond!(cond, self.time, #attr)
            }
        }
    }
    .into()
}

/// Define default handlers trait.
///
/// # Examples
/// ```ignore
/// #[default_handler(users)]
/// #[async_trait]
/// pub trait UserHandler: Send + Sync {
/// }
/// ```
#[proc_macro_attribute]
pub fn default_handler(attr: TokenStream, input: TokenStream) -> TokenStream {
    let attr = parse_macro_input!(attr as Ident);
    let mut input = parse_macro_input!(input as ItemTrait);
    let func: Vec<String> = input
        .items
        .iter()
        .map(|x| {
            if let TraitItem::Fn(x) = x {
                x.sig.ident.to_string()
            } else {
                String::new()
            }
        })
        .collect();
    let contains = |name: &str| func.iter().any(|x| x == name);

    if !contains("find") {
        input.items.push(parse_quote! {
            /// Find record by `cond`.
            async fn find(
                &self,
                db: &sea_orm::DatabaseTransaction,
                cond: Condition,
            ) -> Result<(Vec<#attr::Model>, u64)>;
        });
    }
    if !contains("find_by_id") {
        input.items.push(parse_quote! {
            /// Find record by `id`.
            async fn find_by_id(
                &self,
                db: &sea_orm::DatabaseTransaction,
                id: &HyUuid,
            ) -> Result<Option<#attr::Model>>;
        });
    }
    if !contains("delete_all") {
        input.items.push(parse_quote! {
            /// Delete all records.
            async fn delete_all(&self, db: &sea_orm::DatabaseTransaction) -> Result<u64>;
        });
    }
    if !contains("delete") {
        input.items.push(parse_quote! {
            /// Delete record by `id`.
            async fn delete(&self, db: &sea_orm::DatabaseTransaction, id: &[HyUuid]) -> Result<u64>;
        });
    }
    if !contains("count") {
        input.items.push(parse_quote! {
            /// Count records by `cond`.
            async fn count(
                &self,
                db: &sea_orm::DatabaseTransaction,
                cond: Condition,
            ) -> Result<u64>;
        });
    }

    quote! {
        #input
    }
    .into()
}

/// Implement default handlers.
///
/// # Examples
/// ```ignore
/// #[default_handler_impl(users)]
/// #[async_trait]
/// impl UserHandler for DefaultUserHandler {
/// ...
/// }
/// ```
#[proc_macro_attribute]
pub fn default_handler_impl(attr: TokenStream, input: TokenStream) -> TokenStream {
    let attr = parse_macro_input!(attr as Ident);
    let mut input = parse_macro_input!(input as ItemImpl);
    let func: Vec<String> = input
        .items
        .iter()
        .map(|x| {
            if let ImplItem::Fn(x) = x {
                x.sig.ident.to_string()
            } else {
                String::new()
            }
        })
        .collect();
    let contains = |name: &str| func.iter().any(|x| x == name);
    if !contains("find") {
        input.items.push(parse_quote! {
            async fn find(
                &self,
                db: &skynet_api::sea_orm::DatabaseTransaction,
                cond: skynet_api::request::Condition,
            ) -> actix_cloud::Result<(Vec<#attr::Model>, u64)> {
                cond.select_page(#attr::Entity::find(), db).await
            }
        });
    }
    if !contains("find_by_id") {
        input.items.push(parse_quote! {
            async fn find_by_id(
                &self,
                db: &skynet_api::sea_orm::DatabaseTransaction,
                id: &skynet_api::HyUuid,
            ) -> actix_cloud::Result<Option<#attr::Model>> {
                #attr::Entity::find_by_id(id.to_owned())
                    .one(db)
                    .await
                    .map_err(Into::into)
            }
        });
    }
    if !contains("delete_all") {
        input.items.push(parse_quote! {
            async fn delete_all(
                &self,
                db: &skynet_api::sea_orm::DatabaseTransaction
            ) -> actix_cloud::Result<u64> {
                #attr::Entity::delete_many()
                    .exec(db)
                    .await
                    .map(|x| x.rows_affected)
                    .map_err(Into::into)
            }
        });
    }
    if !contains("delete") {
        input.items.push(parse_quote! {
            async fn delete(
                &self,
                db: &skynet_api::sea_orm::DatabaseTransaction,
                id: &[skynet_api::HyUuid]
            ) -> actix_cloud::Result<u64> {
                #attr::Entity::delete_many()
                    .filter(#attr::Column::Id.is_in(skynet_api::hyuuid::uuids2strings(id)))
                    .exec(db)
                    .await
                    .map(|x| x.rows_affected)
                    .map_err(Into::into)
            }
        });
    }
    if !contains("count") {
        input.items.push(parse_quote! {
            async fn count(
                &self,
                db: &skynet_api::sea_orm::DatabaseTransaction,
                cond: skynet_api::request::Condition,
            ) -> actix_cloud::Result<u64> {
                Ok(cond.build(#attr::Entity::find()).0.count(db).await?)
            }
        });
    }

    quote! {
        #input
    }
    .into()
}

macro_rules! parse_type {
    ($($tt:tt)*) => {{
        let ty: syn::Type = syn::parse_quote! { $($tt)* };
        ty
    }}
}

/// Add span support for plugin API.
/// `skynet::request::Request`/`request::Request`/`Request` will be automatically reused.
///
/// # Examples
/// ```ignore
/// #[plugin_api]
/// async fn get() -> RspResult<JsonResponse> {}
/// // or
/// use skynet::request::Request;
/// #[plugin_api]
/// async fn get_custom(xxx: Request) -> RspResult<JsonResponse> {}
/// ```
#[proc_macro_attribute]
pub fn plugin_api(attr: TokenStream, input: TokenStream) -> TokenStream {
    let attr = parse_macro_input!(attr as Option<Ident>);
    let mut func = parse_macro_input!(input as ItemFn);
    let mut req_name = format_ident!("_plugin_req");
    let mut req_flag = false;
    for i in &func.sig.inputs {
        if let FnArg::Typed(x) = i {
            if *x.ty == parse_type! {skynet_api::request::Request}
                || *x.ty == parse_type! {request::Request}
                || *x.ty == parse_type! {Request}
            {
                if let Pat::Ident(ident) = x.pat.as_ref() {
                    req_name = ident.ident.clone();
                    req_flag = true;
                }
            }
        }
    }
    if !req_flag {
        func.sig
            .inputs
            .insert(0, parse_quote!(_plugin_req: skynet_api::request::Request));
    }
    if let Some(attr) = attr {
        func.block.stmts.insert(
            0,
            parse_quote!(let _plugin_runtime_guard = #attr.get().unwrap().enter();),
        );
    }

    let attrs = func.attrs;
    let vis = func.vis;
    let sig = func.sig;
    let block = func.block;
    let input = TokenStream::from(quote! {
        #(#attrs)*
        #vis #sig {
            async move {
                #block
            }.instrument(_plugin_span).await
        }
    });

    let mut func = parse_macro_input!(input as ItemFn);
    func.block.stmts.insert(
        0,
        parse_quote!(
            let _plugin_span = actix_cloud::tracing::info_span!(
                "HTTP request",
                trace_id = #req_name.extension.trace_id,
                ip = %#req_name.extension.real_ip,
                method = _plugin_method,
                path = _plugin_path,
                user_agent = _plugin_user_agent,
            );
        ),
    );
    func.block.stmts.insert(
        0,
        parse_quote!(
            let _plugin_user_agent = #req_name.http_request
                .headers()
                .get("User-Agent")
                .map_or("", |h| h.to_str().unwrap_or(""))
                .to_owned();
        ),
    );
    func.block.stmts.insert(
        0,
        parse_quote!(
            let _plugin_method = #req_name.http_request.method().to_string();
        ),
    );
    func.block.stmts.insert(
        0,
        parse_quote!(
            let _plugin_path = #req_name.http_request
                .uri()
                .path_and_query()
                .map(ToString::to_string)
                .unwrap_or_default();
        ),
    );

    quote! {
        #func
    }
    .into()
}
