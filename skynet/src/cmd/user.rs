use std::{fs, path::PathBuf};

use actix_cloud::{
    logger::Logger,
    tracing::{debug, error, info},
};
use skynet_api::{
    sea_orm::{ConnectionTrait, TransactionTrait},
    viewer::users::UserViewer,
};

use crate::{Cli, UserCli, UserCommands};

use super::init;

async fn create<C>(db: &C, username: &str, avatar: &Option<PathBuf>, root: bool)
where
    C: ConnectionTrait,
{
    let mut avatar_file: Option<Vec<u8>> = None;
    if let Some(x) = avatar {
        avatar_file = Some(fs::read(x).expect("Read avatar failed"));
        debug!("Read avatar success: {:?}", x);
    }
    if UserViewer::find_by_name(db, username)
        .await
        .unwrap()
        .is_some()
    {
        error!("User `{username}` already exists");
    } else {
        let user = UserViewer::create(db, username, None, avatar_file, root)
            .await
            .unwrap();
        info!("New pass: {}", user.password);
    }
}

pub async fn command(cli: &Cli, logger: Option<Logger>, user_cli: &UserCli) {
    let (skynet, state, db, _, _) = init(cli, logger).await;

    let tx = db.begin().await.unwrap();
    match &user_cli.command {
        // skynet user add
        UserCommands::Add { avatar, username } => create(&tx, username, avatar, false).await,
        // skynet user reset
        UserCommands::Reset { username } => {
            let user = UserViewer::find_by_name(&tx, username).await.unwrap();
            if let Some(x) = user {
                let x = UserViewer::reset(
                    &tx,
                    state.memorydb.as_ref(),
                    &x.id,
                    &skynet.config.session.prefix,
                )
                .await
                .unwrap()
                .unwrap();
                info!("New pass: {}", x.password);
                info!("Reset user success");
            } else {
                error!("User `{username}` not found");
            };
        }
        UserCommands::Init { avatar } => create(&tx, "root", avatar, true).await,
    }
    tx.commit().await.unwrap();
}
