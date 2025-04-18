use sea_orm_migration::prelude::*;

use crate::m20220101_000001_create_table::Users;

#[derive(Iden)]
enum UserHistories {
    Table,
    ID,
    Uid,
    IP,
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
                    .table(UserHistories::Table)
                    .if_not_exists()
                    .col(
                        ColumnDef::new(UserHistories::ID)
                            .char_len(36)
                            .not_null()
                            .primary_key(),
                    )
                    .col(ColumnDef::new(UserHistories::Uid).char_len(36).not_null())
                    .col(ColumnDef::new(UserHistories::IP).string_len(64).not_null())
                    .col(
                        ColumnDef::new(UserHistories::CreatedAt)
                            .big_integer()
                            .not_null(),
                    )
                    .col(
                        ColumnDef::new(UserHistories::UpdatedAt)
                            .big_integer()
                            .not_null(),
                    )
                    .foreign_key(
                        ForeignKey::create()
                            .to(Users::Table, Users::ID)
                            .from_col(UserHistories::Uid)
                            .on_update(ForeignKeyAction::Restrict)
                            .on_delete(ForeignKeyAction::Cascade),
                    )
                    .to_owned(),
            )
            .await?;
        Ok(())
    }

    async fn down(&self, manager: &SchemaManager) -> Result<(), DbErr> {
        manager
            .drop_table(Table::drop().table(UserHistories::Table).to_owned())
            .await?;
        Ok(())
    }
}
