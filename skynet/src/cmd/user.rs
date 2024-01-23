use std::{fs, path::PathBuf};

use log::{debug, error, info};
use sea_orm::{DatabaseConnection, TransactionTrait};
use skynet::Skynet;

use crate::{Cli, UserCli, UserCommands};

use super::run::init_skynet;

async fn create(
    skynet: &Skynet,
    db: &DatabaseConnection,
    avatar: &Option<PathBuf>,
    username: &str,
) {
    let mut avatar_file: Option<Vec<u8>> = None;
    if let Some(x) = avatar {
        avatar_file = Some(fs::read(x).unwrap());
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
            .create(&tx, skynet, username, None, avatar_file)
            .await
            .unwrap();
        info!("New pass: {}", user.password);
    }
    tx.commit().await.unwrap();
}

pub async fn command(cli: &Cli, skynet: Skynet, user_cli: &UserCli) {
    let (skynet, db, redis) = init_skynet(cli, skynet).await;

    match &user_cli.command {
        // skynet user add
        UserCommands::Add { avatar, username } => create(&skynet, &db, avatar, username).await,
        // skynet user reset
        UserCommands::Reset { username } => {
            let tr = db.begin().await.unwrap();
            let user = skynet.user.find_by_name(&tr, username).await.unwrap();
            if let Some(x) = user {
                let x = skynet
                    .user
                    .reset(&tr, &redis, &skynet, &x.id)
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
        UserCommands::Init { avatar } => create(&skynet, &db, avatar, "root").await,
    }
}
