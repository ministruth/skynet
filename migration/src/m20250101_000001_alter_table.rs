use sea_orm_migration::prelude::*;

#[derive(Iden)]
enum UserHistories {
    Table,
    UserAgent,
}

#[derive(DeriveMigrationName)]
pub struct Migration;

#[async_trait::async_trait]
impl MigrationTrait for Migration {
    async fn up(&self, manager: &SchemaManager) -> Result<(), DbErr> {
        manager
            .alter_table(
                Table::alter()
                    .table(UserHistories::Table)
                    .add_column_if_not_exists(
                        ColumnDef::new(UserHistories::UserAgent).string_len(512),
                    )
                    .to_owned(),
            )
            .await?;
        Ok(())
    }

    async fn down(&self, manager: &SchemaManager) -> Result<(), DbErr> {
        manager
            .alter_table(
                Table::alter()
                    .table(UserHistories::Table)
                    .drop_column(UserHistories::UserAgent)
                    .to_owned(),
            )
            .await?;
        Ok(())
    }
}
