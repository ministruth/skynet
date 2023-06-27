import { checkAPI, getIntl, postAPI, UserPerm } from '@/utils';
import { PlusOutlined } from '@ant-design/icons';
import { ParamsType } from '@ant-design/pro-provider';
import { FormattedMessage } from 'react-intl';
import { Columns } from '../layout/table/column';
import TableOp from '../layout/table/opBtn';
import TableBtn from '../layout/table/tableBtn';
import { AvatarColumn, GroupColumn, UserBtnProps, UserColumns } from './card';

const cloneColumns: Columns = (intl) => [
  {
    renderFormItem: () => <FormattedMessage id="pages.user.op.clone.content" />,
  },
  {
    dataIndex: 'base',
    formItemProps: {
      hidden: true,
    },
  },
  {
    title: intl.get('pages.user.form.baseuser'),
    dataIndex: 'baseName',
    readonly: true,
  },
  ...UserColumns(intl),
  AvatarColumn(intl),
  GroupColumn(intl),
  {
    title: intl.get('pages.user.form.clonegroup'),
    dataIndex: 'clone_group',
    valueType: 'switch',
  },
];

const UserClone: React.FC<UserBtnProps> = (props) => {
  const intl = getIntl();
  const handleClone = async (params: ParamsType) => {
    delete params.baseName;
    if (await checkAPI(postAPI('/user', params))) {
      props.tableRef?.current?.reloadAndRest?.();
      return true;
    }
    return false;
  };
  return (
    <TableOp
      trigger={
        <TableBtn
          key="clone"
          icon={PlusOutlined}
          tip={intl.get('pages.user.op.clone.tip')}
        />
      }
      schemaProps={{
        onFinish: handleClone,
        columns: cloneColumns(intl),
        initialValues: props.initialValues,
      }}
      permName="manage.user"
      perm={UserPerm.PermWriteExecute}
      rollback={<PlusOutlined key="clone" />}
      width={500}
      title={intl.get('pages.user.op.clone.title')}
    />
  );
};

export default UserClone;
