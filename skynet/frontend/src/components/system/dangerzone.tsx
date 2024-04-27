import { checkPerm, getIntl, postAPI, StringIntl, UserPerm } from '@/utils';
import ProCard from '@ant-design/pro-card';
import { FormattedMessage, useAccess } from '@umijs/max';
import { Button } from 'antd';
import confirm from '../layout/modal';
import RowItem from '../layout/rowItem';
import styles from './style.less';

const handleShutdown = (intl: StringIntl) => {
  confirm({
    title: intl.get('pages.system.dangerzone.shutdown.title'),
    content: intl.get('pages.system.dangerzone.shutdown.content'),
    onOk() {
      return new Promise((resolve, reject) => {
        postAPI('/shutdown', {}).then((rsp) => {
          if (rsp && rsp.code === 0) resolve(rsp);
          else reject(rsp);
        });
      });
    },
    intl: intl,
  });
};

const DangerZoneCard = () => {
  const intl = getIntl();
  const access = useAccess();

  return (
    <ProCard
      title={intl.get('pages.system.dangerzone.title')}
      className={styles.dangerBorder}
      bordered
    >
      <RowItem
        span={{ xs: 14, md: 6 }}
        text={<FormattedMessage id="pages.system.dangerzone.shutdown.text" />}
        item={
          <Button
            danger
            onClick={() => handleShutdown(intl)}
            disabled={!checkPerm(access, 'manage.system', UserPerm.PermWrite)}
          >
            {intl.get('pages.system.dangerzone.shutdown.button')}
          </Button>
        }
        nospace
      ></RowItem>
    </ProCard>
  );
};

export default DangerZoneCard;
