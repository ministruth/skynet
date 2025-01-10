import { getAPI, getIntl, postAPI, StringIntl } from '@/utils';
import { ParamsType } from '@ant-design/pro-components';
import { ProColumns } from '@ant-design/pro-table';
import { FormattedMessage } from '@umijs/max';
import { Button } from 'antd';
import { SortOrder } from 'antd/es/table/interface';
import confirm from '../layout/modal';
import Table from '../layout/table';
import UserAgent from '../user/userAgent';
import styles from './style.less';

const request = async (
  params?: ParamsType,
  _sort?: Record<string, SortOrder>,
) => {
  const msg = await getAPI('/users/self/sessions', {
    page: params?.current,
    size: params?.pageSize,
  });
  return {
    data: msg.data.data,
    success: true,
    total: msg.data.total,
  };
};

const handleKick = (intl: StringIntl) => {
  confirm({
    title: intl.get('pages.account.kick.title'),
    content: intl.get('app.confirm'),
    onOk() {
      return new Promise((resolve, reject) => {
        postAPI(`/users/self/kick`, {}).then((rsp) => {
          if (rsp && rsp.code === 0) {
            resolve(rsp);
            window.location.href = '/';
          } else {
            reject(rsp);
          }
        });
      });
    },
    intl: intl,
  });
};

const SessionList = () => {
  const intl = getIntl();

  const columns: ProColumns[] = [
    {
      title: intl.get('tables.time'),
      dataIndex: 'time',
      align: 'center',
      valueType: 'dateTime',
      hideInSearch: true,
    },
    {
      title: intl.get('tables.device'),
      dataIndex: 'user_agent',
      align: 'center',
      hideInSearch: true,
      render: (_, row) => <UserAgent value={row.user_agent} />,
    },
    {
      title: intl.get('tables.ttl'),
      dataIndex: 'ttl',
      align: 'center',
      hideInSearch: true,
    },
  ];

  return (
    <Table
      rowKey="name"
      className={styles['column-padding-end']}
      headerTitle={intl.get('pages.account.session.title')}
      options={false}
      search={false}
      columns={columns}
      request={request}
      cardBordered={false}
      action={[
        <Button key="kick" danger onClick={() => handleKick(intl)}>
          <FormattedMessage id="pages.account.kick" />
        </Button>,
      ]}
    />
  );
};

export default SessionList;
