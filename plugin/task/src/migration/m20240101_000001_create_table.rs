use sea_orm_migration::{MigrationTrait, SchemaManager};
use skynet_api::{
    async_trait,
    sea_orm::{
        sea_query::{self, ColumnDef, Iden, Table},
        DbErr, DeriveMigrationName,
    },
};

use super::migrator::table_prefix;

#[derive(Iden)]
enum Tasks {
    Table,
    ID,
    Name,
    Detail,
    Output,
    Result,
    Percent,
    CreatedAt,
    UpdatedAt,
}

#[derive(DeriveMigrationName)]
pub struct Migration;

#[async_trait]
impl MigrationTrait for Migration {
    async fn up(&self, manager: &SchemaManager) -> Result<(), DbErr> {
        manager
            .create_table(
                Table::create()
                    .table(table_prefix(&Tasks::Table))
                    .if_not_exists()
                    .col(
                        ColumnDef::new(Tasks::ID)
                            .char_len(36)
                            .not_null()
                            .primary_key(),
                    )
                    .col(ColumnDef::new(Tasks::Name).string_len(64).not_null())
                    .col(ColumnDef::new(Tasks::Detail).string_len(1024))
                    .col(ColumnDef::new(Tasks::Output).string())
                    .col(ColumnDef::new(Tasks::Result).integer())
                    .col(
                        ColumnDef::new(Tasks::Percent)
                            .integer()
                            .default(0)
                            .not_null(),
                    )
                    .col(ColumnDef::new(Tasks::CreatedAt).big_integer().not_null())
                    .col(ColumnDef::new(Tasks::UpdatedAt).big_integer().not_null())
                    .to_owned(),
            )
            .await?;
        Ok(())
    }

    async fn down(&self, manager: &SchemaManager) -> Result<(), DbErr> {
        manager
            .drop_table(Table::drop().table(table_prefix(&Tasks::Table)).to_owned())
            .await?;
        Ok(())
    }
}
