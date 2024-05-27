use proc_macro::TokenStream;
use quote::{format_ident, quote};
use syn::{
    parse::Parser, parse_macro_input, parse_quote, punctuated::Punctuated, Data, DeriveInput, Expr,
    ExprPath, Fields, FnArg, Ident, ImplItem, ItemFn, ItemImpl, ItemStruct, ItemTrait, Pat, Token,
    TraitItem,
};

/// Default timestamp generator.
///
/// Automatically generate `created_at` and `updated_at` on create and update.
///
/// # Examples
/// ```ignore
/// #[entity_timestamp]
/// impl ActiveModel {}
/// ```
#[proc_macro_attribute]
pub fn entity_timestamp(_: TokenStream, input: TokenStream) -> TokenStream {
    let mut entity = parse_macro_input!(input as ItemImpl);
    entity.items.push(parse_quote!(
        fn entity_timestamp(&self, e: &mut Self, insert: bool) {
            let tm: sea_orm::ActiveValue<i64> =
                sea_orm::ActiveValue::set(crate::Utc::now().timestamp_millis());
            if insert {
                e.created_at = tm.clone();
                e.updated_at = tm.clone();
            } else {
                e.updated_at = tm.clone();
            }
        }
    ));
    quote! {
        #entity
    }
    .into()
}

/// Default id generator.
///
/// Automatically generate `id` on create.
///
/// # Examples
/// ```ignore
/// #[entity_id]
/// impl ActiveModel {}
/// ```
#[proc_macro_attribute]
pub fn entity_id(_: TokenStream, input: TokenStream) -> TokenStream {
    let mut entity = parse_macro_input!(input as ItemImpl);
    entity.items.push(parse_quote!(
        fn entity_id(&self, e: &mut Self, insert: bool) {
            if insert && e.id.is_not_set() {
                e.id = sea_orm::ActiveValue::set(crate::HyUuid::new());
            }
        }
    ));
    quote! {
        #entity
    }
    .into()
}

/// Default entity behavior:
/// - `entity_id`
/// - `entity_timestamp`
///
/// # Examples
/// ```ignore
/// #[entity_behavior]
/// impl ActiveModelBehavior for ActiveModel {}
/// ```
#[proc_macro_attribute]
pub fn entity_behavior(_: TokenStream, input: TokenStream) -> TokenStream {
    let mut entity = parse_macro_input!(input as ItemImpl);

    entity.items.push(parse_quote!(
        async fn before_save<C>(self, _: &C, insert: bool) -> Result<Self, DbErr>
        where
            C: ConnectionTrait,
        {
            let mut new = self.clone();
            self.entity_id(&mut new, insert);
            self.entity_timestamp(&mut new, insert);
            Ok(new)
        }
    ));
    quote! {
        #[async_trait::async_trait]
        #entity
    }
    .into()
}

/// Implement `into` for entity to partial entity.
/// The fields should be exactly the same.
///
/// # Examples
/// ```ignore
/// #[partial_entity(users::Model)]
/// #[derive(Serialize)]
/// struct Rsp {
///     pub id: HyUuid,
///     ...
/// }
///
/// let y: users::Model = ...;
/// let x: Rsp = y.into();
/// ```
#[proc_macro_attribute]
pub fn partial_entity(attr: TokenStream, input: TokenStream) -> TokenStream {
    let attr = parse_macro_input!(attr as ExprPath);
    let input = parse_macro_input!(input as ItemStruct);
    let name = &input.ident;
    let mut fields = Vec::new();
    for i in &input.fields {
        let field_name = &i.ident;
        fields.push(quote!(#field_name: self.#field_name,));
    }

    quote! {
        #input
        impl Into<#name> for #attr {
            fn into(self) -> #name {
                #name {
                    #(#fields)*
                }
            }
        }
    }
    .into()
}

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
    let attr = parse_macro_input!(attr as Ident);
    let input = parse_macro_input!(input as ItemStruct);
    let name = &input.ident;

    quote! {
        #input
        impl #name {
            pub fn common_cond(&self) -> skynet::Condition {
                let mut cond = skynet::Condition::new(sea_orm::Condition::all()).add_page(self.page.clone());
                skynet::build_time_cond!(cond, self.time, #attr)
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
                db: &sea_orm::DatabaseTransaction,
                cond: skynet::Condition,
            ) -> Result<(Vec<#attr::Model>, u64)> {
                let (q, page) = cond.build(#attr::Entity::find());
                if let Some(page) = page {
                    let q = q.paginate(db, page.size);
                    Ok((q.fetch_page(page.page - 1).await?, q.num_items().await?))
                } else {
                    let res = q.all(db).await?;
                    let cnt = res.len() as u64;
                    Ok((res, cnt))
                }
            }
        });
    }
    if !contains("find_by_id") {
        input.items.push(parse_quote! {
            async fn find_by_id(
                &self,
                db: &sea_orm::DatabaseTransaction,
                id: &skynet::HyUuid,
            ) -> Result<Option<#attr::Model>> {
                #attr::Entity::find_by_id(id.to_owned())
                    .one(db)
                    .await
                    .map_err(anyhow::Error::from)
            }
        });
    }
    if !contains("delete_all") {
        input.items.push(parse_quote! {
            async fn delete_all(&self, db: &sea_orm::DatabaseTransaction) -> Result<u64> {
                #attr::Entity::delete_many()
                    .exec(db)
                    .await
                    .map(|x| x.rows_affected)
                    .map_err(anyhow::Error::from)
            }
        });
    }
    if !contains("delete") {
        input.items.push(parse_quote! {
            async fn delete(&self, db: &sea_orm::DatabaseTransaction, id: &[skynet::HyUuid]) -> Result<u64> {
                #attr::Entity::delete_many()
                    .filter(#attr::Column::Id.is_in(skynet::hyuuid::uuid2string(id)))
                    .exec(db)
                    .await
                    .map(|x| x.rows_affected)
                    .map_err(anyhow::Error::from)
        }
    });
    }
    if !contains("count") {
        input.items.push(parse_quote! {
            async fn count(
                &self,
                db: &sea_orm::DatabaseTransaction,
                cond: skynet::Condition,
            ) -> Result<u64> {
                Ok(cond.build(#attr::Entity::find()).0.count(db).await?)
            }
        });
    }

    quote! {
        #input
    }
    .into()
}

/// Generate `foreach_StructName` macro of the struct.
///
/// # Examples
///
/// ```ignore
/// #[derive(Default, Foreach)]
/// pub struct Config {
///     pub fields1: Option<i32>,
///     pub fields2: Option<String>,
/// }
///
/// let cfg = Config::default();
/// foreach_Config!(
///     cfg, v,
///     if let Some(x) = v {
///         println!("{:?}", v);
///     }
/// );
/// ```
///
/// # Panics
///
/// Panics when not applied to structs with named fields.
#[proc_macro_derive(Foreach)]
pub fn foreach(input: TokenStream) -> TokenStream {
    let input = parse_macro_input!(input as DeriveInput);
    let macro_name = format_ident!("foreach_{}", &input.ident);

    let fields = match input.data {
        Data::Struct(data_struct) => match data_struct.fields {
            Fields::Named(fields_named) => fields_named.named,
            _ => panic!("Only structs with named fields are supported"),
        },
        _ => panic!("Only structs are supported"),
    };

    let fields_iter = fields.iter().map(|field| {
        let field_ident = &field.ident;
        quote! {
            let $v = &$c.#field_ident;
            $($stmt)*
        }
    });

    let x = quote! {
        #[macro_export]
        macro_rules! #macro_name {
            ($c:expr,$v:ident,$($stmt:stmt)*) => {
                {
                    #(#fields_iter)*
                }
            };
        }
    };
    x.into()
}

/// The `Iterable` proc macro.
///
/// Deriving this macro for your struct will make it "iterable". An iterable struct allows you to iterate over its fields, returning a tuple containing the field name as a static string and a reference to the field's value as `dyn Any`.
///
/// # Limitations
///
/// - Only structs are supported, not enums or unions.
/// - Only structs with named fields are supported.
///
/// # Usage
///
/// Add the derive attribute (`#[derive(Iterable)]`) above your struct definition.
///
/// ```ignore
/// use skynet_macro::Iterable;
///
/// #[derive(Iterable)]
/// struct MyStruct {
///     field1: i32,
///     field2: String,
/// }
/// ```
///
/// You can now call the `iter` method on instances of your struct to get an iterator over its fields:
///
/// ```ignore
/// let my_instance = MyStruct {
///     field1: 42,
///     field2: "Hello, world!".to_string(),
/// };
///
/// for (field_name, field_value) in my_instance.iter() {
///     println!("{}: {:?}", field_name, field_value);
/// }
/// ```
///
/// Or call the `iter_mut` method to modify the fields:
///
/// ```ignore
/// let mut my_instance = MyStruct {
///     field1: 42,
///     field2: "Hello, world!".to_string(),
/// };
///
/// for (field_name, field_value) in my_instance.iter_mut() {
///     if let Some(num) = field_value.downcast_mut::<i32>() {
///         *num += 1;
///     }
/// }
/// ```
///
/// # Panics
///
/// Panics when not applied to structs with named fields.
#[proc_macro_derive(Iterable)]
pub fn derive_iterable(input: TokenStream) -> TokenStream {
    let input = parse_macro_input!(input as DeriveInput);

    let struct_name = input.ident;
    let fields = match input.data {
        Data::Struct(data_struct) => match data_struct.fields {
            Fields::Named(fields_named) => fields_named.named,
            _ => panic!("Only structs with named fields are supported"),
        },
        _ => panic!("Only structs are supported"),
    };

    let fields_iter = fields.iter().map(|field| {
        let field_ident = &field.ident;
        let field_name = field_ident.as_ref().unwrap().to_string();
        quote! {
            (#field_name, &(self.#field_ident) as &dyn std::any::Any)
        }
    });

    let fields_iter_mut = fields.iter().map(|field| {
        let field_ident = &field.ident;
        let field_name = field_ident.as_ref().unwrap().to_string();
        quote! {
            (#field_name, &mut (self.#field_ident) as &mut dyn std::any::Any)
        }
    });

    let expanded = quote! {
        impl #struct_name {

            #[allow(clippy::iter_without_into_iter)]
            pub fn iter<'a>(&'a self) -> std::vec::IntoIter<(&'static str, &'a dyn std::any::Any)> {
                vec![
                    #(#fields_iter),*
                ].into_iter()
            }

            #[allow(clippy::iter_without_into_iter)]
            pub fn iter_mut<'a>(&'a mut self) -> std::vec::IntoIter<(&'static str, &'a mut dyn std::any::Any)> {
                vec![
                    #(#fields_iter_mut),*
                ].into_iter()
            }

        }
    };

    TokenStream::from(expanded)
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
/// async fn get() -> RspResult<impl Responder> {}
/// // or
/// use skynet::request::Request;
/// #[plugin_api]
/// async fn get_custom(xxx: Request) -> RspResult<impl Responder> {}
/// ```
#[proc_macro_attribute]
pub fn plugin_api(attr: TokenStream, input: TokenStream) -> TokenStream {
    let attr = parse_macro_input!(attr as Option<Ident>);
    let mut func = parse_macro_input!(input as ItemFn);
    let mut name = format_ident!("_plugin_span_req");
    let mut flag = false;
    for i in &func.sig.inputs {
        if let FnArg::Typed(x) = i {
            if *x.ty == parse_type! {skynet::request::Request}
                || *x.ty == parse_type! {request::Request}
                || *x.ty == parse_type! {Request}
            {
                if let Pat::Ident(ident) = x.pat.as_ref() {
                    name = ident.ident.clone();
                    flag = true;
                    break;
                }
            }
        }
    }
    if !flag {
        func.sig
            .inputs
            .insert(0, parse_quote!(_plugin_span_req: skynet::request::Request));
    }
    if let Some(attr) = attr {
        func.block.stmts.insert(
            0,
            parse_quote!(let _skynet_plugin_runtime_guard = #attr.get().unwrap().enter();),
        );
    }
    func.block.stmts.insert(
        0,
        parse_quote!(let _skynet_plugin_span_guard =
        skynet::tracing::info_span!("HTTP request", request_id = %#name.request_id, ip = %#name.ip)
            .entered();),
    );

    quote! {
        #func
    }
    .into()
}

/// Add span support for the API.
///
/// # Examples
/// ```ignore
/// #[tracing_api(self.request_id, self.ip)]
/// fn test(&self) {}
/// ```
///
/// # Panics
///
/// Panics when not arguments are incorrect.
#[proc_macro_attribute]
pub fn tracing_api(attr: TokenStream, input: TokenStream) -> TokenStream {
    let args = Punctuated::<Expr, Token![,]>::parse_terminated
        .parse(attr)
        .unwrap();
    assert!(args.len() == 2, "arguments must be 2");
    let request_id = args.first();
    let ip = args.last();
    let mut func = parse_macro_input!(input as ItemFn);
    func.block.stmts.insert(
        0,
        parse_quote!(let _skynet_span_guard = skynet::tracing::info_span!("API call", request_id = %#request_id, ip = %#ip).entered();),
    );

    quote! {
        #func
    }
    .into()
}
