import { getIntl, UserPerm } from '@/utils';
import { ProColumns } from '@ant-design/pro-table';
import { FormattedMessage, useAccess } from '@umijs/max';
import { Tag } from 'antd';
import _ from 'lodash';
import Table from '../layout/table';
import styles from './style.less';

const PermissionList = () => {
  const intl = getIntl();
  const access = useAccess();
  const data = _.sortBy(
    Object.entries(access).map(([key, value]) => ({
      name: key,
      perm: value,
    })),
    'name',
  );

  const columns: ProColumns[] = [
    {
      title: intl.get('tables.name'),
      dataIndex: 'name',
      align: 'center',
      ellipsis: true,
    },
    {
      title: intl.get('tables.perm'),
      dataIndex: 'perm',
      align: 'center',
      render: (_, row) => {
        let item = [];
        if (row.perm == UserPerm.PermNone)
          item.push(
            <Tag>
              <FormattedMessage id="tables.ban" />
            </Tag>,
          );
        if ((row.perm & UserPerm.PermRead) == UserPerm.PermRead)
          item.push(
            <Tag color="blue">
              <FormattedMessage id="tables.read" />
            </Tag>,
          );
        if ((row.perm & UserPerm.PermWrite) == UserPerm.PermWrite)
          item.push(
            <Tag color="volcano">
              <FormattedMessage id="tables.write" />
            </Tag>,
          );
        return <>{item}</>;
      },
    },
  ];

  return (
    <Table
      rowKey="name"
      className={styles['column-padding-start']}
      headerTitle={intl.get('pages.account.permission.title')}
      options={false}
      search={false}
      dataSource={data}
      columns={columns}
      cardBordered={false}
      pagination={{
        defaultPageSize: 5,
      }}
    />
  );
};

export default PermissionList;
