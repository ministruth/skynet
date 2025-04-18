use sea_orm_migration::prelude::*;

use crate::m20220101_000001_create_table::Users;

#[derive(Iden)]
enum WebpushClients {
    Table,
    ID,
    Uid,
    Endpoint,
    P256dh,
    Auth,
    Lang,
    CreatedAt,
    UpdatedAt,
}

#[derive(Iden)]
enum WebpushSubscriptions {
    Table,
    ID,
    Uid,
    Topic,
    CreatedAt,
    UpdatedAt,
}

#[derive(DeriveMigrationName)]
pub struct Migration;

#[async_trait::async_trait]
impl MigrationTrait for Migration {
    async fn up(&self, manager: &SchemaManager) -> Result<(), DbErr> {
        manager
            .create_table(
                Table::create()
                    .table(WebpushClients::Table)
                    .if_not_exists()
                    .col(
                        ColumnDef::new(WebpushClients::ID)
                            .char_len(36)
                            .not_null()
                            .primary_key(),
                    )
                    .col(ColumnDef::new(WebpushClients::Uid).char_len(36).not_null())
                    .col(
                        ColumnDef::new(WebpushClients::Endpoint)
                            .string()
                            .not_null()
                            .unique_key(),
                    )
                    .col(ColumnDef::new(WebpushClients::P256dh).string().not_null())
                    .col(ColumnDef::new(WebpushClients::Auth).string().not_null())
                    .col(
                        ColumnDef::new(WebpushClients::Lang)
                            .string_len(16)
                            .not_null(),
                    )
                    .col(
                        ColumnDef::new(WebpushClients::CreatedAt)
                            .big_integer()
                            .not_null(),
                    )
                    .col(
                        ColumnDef::new(WebpushClients::UpdatedAt)
                            .big_integer()
                            .not_null(),
                    )
                    .foreign_key(
                        ForeignKey::create()
                            .to(Users::Table, Users::ID)
                            .from_col(WebpushClients::Uid)
                            .on_update(ForeignKeyAction::Restrict)
                            .on_delete(ForeignKeyAction::Cascade),
                    )
                    .to_owned(),
            )
            .await?;
        manager
            .create_table(
                Table::create()
                    .table(WebpushSubscriptions::Table)
                    .if_not_exists()
                    .col(
                        ColumnDef::new(WebpushSubscriptions::ID)
                            .char_len(36)
                            .not_null()
                            .primary_key(),
                    )
                    .col(
                        ColumnDef::new(WebpushSubscriptions::Uid)
                            .char_len(36)
                            .not_null(),
                    )
                    .col(
                        ColumnDef::new(WebpushSubscriptions::Topic)
                            .char_len(36)
                            .not_null(),
                    )
                    .col(
                        ColumnDef::new(WebpushSubscriptions::CreatedAt)
                            .big_integer()
                            .not_null(),
                    )
                    .col(
                        ColumnDef::new(WebpushSubscriptions::UpdatedAt)
                            .big_integer()
                            .not_null(),
                    )
                    .foreign_key(
                        ForeignKey::create()
                            .to(Users::Table, Users::ID)
                            .from_col(WebpushSubscriptions::Uid)
                            .on_update(ForeignKeyAction::Restrict)
                            .on_delete(ForeignKeyAction::Cascade),
                    )
                    .to_owned(),
            )
            .await?;
        manager
            .create_index(
                Index::create()
                    .unique()
                    .name("idx_webpushsubscriptions_1")
                    .table(WebpushSubscriptions::Table)
                    .col(WebpushSubscriptions::Uid)
                    .col(WebpushSubscriptions::Topic)
                    .to_owned(),
            )
            .await?;
        Ok(())
    }

    async fn down(&self, manager: &SchemaManager) -> Result<(), DbErr> {
        manager
            .drop_table(Table::drop().table(WebpushClients::Table).to_owned())
            .await?;
        Ok(())
    }
}
