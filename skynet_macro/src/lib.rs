use proc_macro::TokenStream;
use quote::quote;
use syn::{ExprPath, Ident, ImplItem, ItemImpl, ItemStruct, parse_macro_input, parse_quote};

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

/// Implement default viewer.
///
/// # Examples
/// ```ignore
/// pub struct UserViewer;
///
/// #[default_viewer]
/// impl UserViewer {}
/// ```
#[proc_macro_attribute]
pub fn default_viewer(attr: TokenStream, input: TokenStream) -> TokenStream {
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
            pub async fn find<C>(
                db: &C,
                cond: Condition,
            ) -> anyhow::Result<(Vec<#attr::Model>, u64)>
            where
                C: sea_orm::ConnectionTrait
            {
                cond.select_page(#attr::Entity::find(), db).await
            }
        });
    }
    if !contains("find_by_id") {
        input.items.push(parse_quote! {
            pub async fn find_by_id<C>(db: &C, id: &HyUuid) -> anyhow::Result<Option<#attr::Model>>
            where
                C: sea_orm::ConnectionTrait,
            {
                #attr::Entity::find_by_id(id.to_owned())
                    .one(db)
                    .await
                    .map_err(Into::into)
            }
        });
    }
    if !contains("delete_all") {
        input.items.push(parse_quote! {
            pub async fn delete_all<C>(db: &C) -> anyhow::Result<u64>
            where
                C: sea_orm::ConnectionTrait,
            {
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
            pub async fn delete<C>(db: &C, id: &[HyUuid]) -> anyhow::Result<u64>
            where
                C: sea_orm::ConnectionTrait,
            {
                #attr::Entity::delete_many()
                    .filter(#attr::Column::Id.is_in(uuids2strings(id)))
                    .exec(db)
                    .await
                    .map(|x| x.rows_affected)
                    .map_err(Into::into)
            }
        });
    }
    if !contains("count") {
        input.items.push(parse_quote! {
            pub async fn count<C>(
                db: &C,
                cond: Condition,
            ) -> anyhow::Result<u64>
            where
                C: sea_orm::ConnectionTrait,
            {
                Ok(cond.build(#attr::Entity::find()).0.count(db).await?)
            }
        });
    }

    quote! {
        #input
    }
    .into()
}
