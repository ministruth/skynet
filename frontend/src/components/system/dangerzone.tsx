import {
  checkPerm,
  getIntl,
  ping,
  postAPI,
  StringIntl,
  UserPerm,
} from '@/utils';
import ProCard from '@ant-design/pro-card';
import { Button } from 'antd';
import { FormattedMessage, history, useAccess, useModel } from 'umi';
import confirm from '../layout/modal';
import RowItem from '../layout/rowitem';
import styles from './style.less';

const handleReload = (intl: StringIntl, refresh: () => Promise<void>) => {
  let t: NodeJS.Timer;
  confirm({
    title: intl.get('pages.system.dangerzone.reload.title'),
    content: intl.get('pages.system.dangerzone.reload.content'),
    onOk() {
      return new Promise((resolve, reject) => {
        postAPI('/reload', {}).then(async (rsp) => {
          if (rsp && rsp.code === 0)
            t = setInterval(async () => {
              if (await ping()) {
                clearInterval(t);
                resolve(rsp);
                refresh().then(() => {
                  history.push('/');
                });
              }
            }, 1000);
          else reject(rsp);
        });
      });
    },
    intl: intl,
  });
};

const DangerZoneCard = () => {
  const intl = getIntl();
  const { initialState, refresh } = useModel('@@initialState');
  const access = useAccess();

  return (
    <ProCard
      title={intl.get('pages.system.dangerzone.title')}
      className={styles.dangerBorder}
      bordered
    >
      <RowItem
        span={3}
        text={<FormattedMessage id="pages.system.dangerzone.reload.text" />}
        item={
          <Button
            danger
            onClick={() => handleReload(intl, refresh)}
            disabled={
              !checkPerm(
                initialState?.signin,
                access,
                'manage.system',
                UserPerm.PermExecute,
              )
            }
          >
            {intl.get('pages.system.dangerzone.reload.button')}
          </Button>
        }
        nospace
      ></RowItem>
    </ProCard>
  );
};

export default DangerZoneCard;
