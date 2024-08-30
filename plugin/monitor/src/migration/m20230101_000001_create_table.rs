use sea_orm_migration::{MigrationTrait, SchemaManager};
use skynet_api::{
    async_trait,
    sea_orm::{
        sea_query::{self, ColumnDef, ForeignKey, ForeignKeyAction, Iden, Index, Table},
        DbErr, DeriveMigrationName,
    },
};

use super::migrator::table_prefix;

#[derive(Iden)]
enum Agents {
    Table,
    ID,
    Uid,
    Name,
    OS,
    Hostname,
    IP,
    System,
    Arch,
    LastLogin,
    CreatedAt,
    UpdatedAt,
}

#[derive(Iden)]
enum AgentSettings {
    Table,
    ID,
    Aid,
    Name,
    Value,
    CreatedAt,
    UpdatedAt,
}

#[derive(Iden)]
enum PassiveAgents {
    Table,
    ID,
    Name,
    Address,
    RetryTime,
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
                    .table(table_prefix(&Agents::Table))
                    .if_not_exists()
                    .col(
                        ColumnDef::new(Agents::ID)
                            .char_len(36)
                            .not_null()
                            .primary_key(),
                    )
                    .col(
                        ColumnDef::new(Agents::Uid)
                            .char_len(32)
                            .unique_key()
                            .not_null(),
                    )
                    .col(ColumnDef::new(Agents::Name).string_len(32).not_null())
                    .col(ColumnDef::new(Agents::OS).string_len(32))
                    .col(ColumnDef::new(Agents::Hostname).string_len(256))
                    .col(ColumnDef::new(Agents::IP).string_len(64).not_null())
                    .col(ColumnDef::new(Agents::System).string_len(128))
                    .col(ColumnDef::new(Agents::Arch).string_len(32))
                    .col(ColumnDef::new(Agents::LastLogin).big_integer().not_null())
                    .col(ColumnDef::new(Agents::CreatedAt).big_integer().not_null())
                    .col(ColumnDef::new(Agents::UpdatedAt).big_integer().not_null())
                    .to_owned(),
            )
            .await?;
        manager
            .create_table(
                Table::create()
                    .table(table_prefix(&AgentSettings::Table))
                    .if_not_exists()
                    .col(
                        ColumnDef::new(AgentSettings::ID)
                            .char_len(36)
                            .not_null()
                            .primary_key(),
                    )
                    .col(ColumnDef::new(AgentSettings::Aid).char_len(36).not_null())
                    .col(
                        ColumnDef::new(AgentSettings::Name)
                            .string_len(256)
                            .not_null(),
                    )
                    .col(ColumnDef::new(AgentSettings::Value).string().not_null())
                    .col(
                        ColumnDef::new(AgentSettings::CreatedAt)
                            .big_integer()
                            .not_null(),
                    )
                    .col(
                        ColumnDef::new(AgentSettings::UpdatedAt)
                            .big_integer()
                            .not_null(),
                    )
                    .foreign_key(
                        ForeignKey::create()
                            .to(table_prefix(&Agents::Table), Agents::ID)
                            .from_col(AgentSettings::Aid)
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
                    .name("idx_agentsettings_1")
                    .table(table_prefix(&AgentSettings::Table))
                    .col(AgentSettings::Aid)
                    .col(AgentSettings::Name)
                    .to_owned(),
            )
            .await?;
        manager
            .create_table(
                Table::create()
                    .table(table_prefix(&PassiveAgents::Table))
                    .if_not_exists()
                    .col(
                        ColumnDef::new(PassiveAgents::ID)
                            .char_len(36)
                            .not_null()
                            .primary_key(),
                    )
                    .col(
                        ColumnDef::new(PassiveAgents::Name)
                            .string_len(32)
                            .not_null()
                            .unique_key(),
                    )
                    .col(
                        ColumnDef::new(PassiveAgents::Address)
                            .string_len(64)
                            .not_null()
                            .unique_key(),
                    )
                    .col(
                        ColumnDef::new(PassiveAgents::RetryTime)
                            .integer()
                            .not_null(),
                    )
                    .col(
                        ColumnDef::new(PassiveAgents::CreatedAt)
                            .big_integer()
                            .not_null(),
                    )
                    .col(
                        ColumnDef::new(PassiveAgents::UpdatedAt)
                            .big_integer()
                            .not_null(),
                    )
                    .to_owned(),
            )
            .await?;
        Ok(())
    }

    async fn down(&self, manager: &SchemaManager) -> Result<(), DbErr> {
        manager
            .drop_table(Table::drop().table(table_prefix(&Agents::Table)).to_owned())
            .await?;
        manager
            .drop_table(
                Table::drop()
                    .table(table_prefix(&AgentSettings::Table))
                    .to_owned(),
            )
            .await?;
        manager
            .drop_table(
                Table::drop()
                    .table(table_prefix(&PassiveAgents::Table))
                    .to_owned(),
            )
            .await?;
        Ok(())
    }
}
