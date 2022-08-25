import { checkAPI, getIntl, postAPI, UserPerm } from '@/utils';
import { PlusOutlined } from '@ant-design/icons';
import { ParamsType } from '@ant-design/pro-provider';
import { FormattedMessage } from 'react-intl';
import { Columns } from '../layout/table/column';
import TableOp from '../layout/table/opBtn';
import TableBtn from '../layout/table/tipBtn';
import { GroupBtnProps, GroupColumns } from './card';

const cloneColumns: Columns = (intl) => [
  {
    renderFormItem: () => (
      <FormattedMessage id="pages.group.table.clone.content" />
    ),
  },
  {
    dataIndex: 'base',
    formItemProps: {
      hidden: true,
    },
  },
  {
    title: intl.get('pages.group.table.clone.base'),
    dataIndex: 'baseName',
    readonly: true,
  },
  ...GroupColumns(intl),
  {
    title: intl.get('pages.group.table.clone.user'),
    dataIndex: 'clone_user',
    valueType: 'switch',
  },
];

const handleClone = (params: ParamsType) => {
  delete params.baseName;
  return checkAPI(postAPI('/group', params));
};

const GroupClone: React.FC<GroupBtnProps> = (props) => {
  const intl = getIntl();
  return (
    <TableOp
      trigger={
        <TableBtn
          key="clone"
          icon={PlusOutlined}
          tip={intl.get('pages.group.table.clonetip')}
        />
      }
      permName="manage.group"
      perm={UserPerm.PermWriteExecute}
      rollback={<PlusOutlined key="clone" />}
      finish={handleClone}
      width={500}
      title={intl.get('pages.group.table.clone')}
      columns={cloneColumns(intl)}
      {...props}
    />
  );
};

export default GroupClone;
