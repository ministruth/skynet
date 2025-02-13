#![allow(clippy::too_many_lines)]
use sea_orm_migration::prelude::*;

mod m20220101_000001_create_table;
mod m20241201_000001_create_table;
mod m20250101_000001_alter_table;
mod m20250119_000001_create_table;

pub struct Migrator;

#[async_trait::async_trait]
impl MigratorTrait for Migrator {
    fn migrations() -> Vec<Box<dyn MigrationTrait>> {
        vec![
            Box::new(m20220101_000001_create_table::Migration),
            Box::new(m20241201_000001_create_table::Migration),
            Box::new(m20250101_000001_alter_table::Migration),
            Box::new(m20250119_000001_create_table::Migration),
        ]
    }
}
