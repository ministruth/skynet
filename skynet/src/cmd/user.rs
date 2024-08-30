use std::{fs, path::PathBuf};

use skynet_api::{
    logger::Logger,
    sea_orm::{DatabaseConnection, TransactionTrait},
    tracing::{debug, error, info},
    Skynet,
};

use crate::{Cli, UserCli, UserCommands};

use super::init;

async fn create(
    skynet: &Skynet,
    db: &DatabaseConnection,
    avatar: &Option<PathBuf>,
    username: &str,
    root: bool,
) {
    let mut avatar_file: Option<Vec<u8>> = None;
    if let Some(x) = avatar {
        avatar_file = Some(fs::read(x).expect("Read avatar failed"));
        debug!("Read avatar success: {:?}", x);
    }
    let tx = db.begin().await.unwrap();
    if skynet
        .user
        .find_by_name(&tx, username)
        .await
        .unwrap()
        .is_some()
    {
        error!("User `{username}` already exists");
    } else {
        let user = skynet
            .user
            .create(&tx, username, None, avatar_file, root)
            .await
            .unwrap();
        info!("New pass: {}", user.password);
    }
    tx.commit().await.unwrap();
}

pub async fn command(cli: &Cli, logger: Logger, user_cli: &UserCli) {
    let (skynet, state, db) = init(cli, logger).await;

    match &user_cli.command {
        // skynet user add
        UserCommands::Add { avatar, username } => {
            create(&skynet, &db, avatar, username, false).await
        }
        // skynet user reset
        UserCommands::Reset { username } => {
            let tr = db.begin().await.unwrap();
            let user = skynet.user.find_by_name(&tr, username).await.unwrap();
            if let Some(x) = user {
                let x = skynet
                    .user
                    .reset(&tr, state.memorydb, &skynet, &x.id)
                    .await
                    .unwrap()
                    .unwrap();
                info!("New pass: {}", x.password);
                info!("Reset user success");
            } else {
                error!("User `{username}` not found");
            };
            tr.commit().await.unwrap();
        }
        UserCommands::Init { avatar } => create(&skynet, &db, avatar, "root", true).await,
    }
}
