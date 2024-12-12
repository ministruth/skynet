import { UserPerm, checkAPI, getIntl, postAPI } from '@/utils';
import { PlusOutlined } from '@ant-design/icons';
import { ParamsType } from '@ant-design/pro-provider';
import { Alert } from 'antd';
import { Columns } from '../layout/table/column';
import TableOp from '../layout/table/opBtn';
import TableBtn from '../layout/table/tableBtn';
import { GroupBtnProps, GroupColumns } from './card';

const cloneColumns: Columns = (intl) => [
  {
    renderFormItem: () => (
      <Alert
        message={intl.get('pages.group.clone.content')}
        type="info"
        showIcon
      />
    ),
  },
  {
    dataIndex: 'base',
    formItemProps: {
      hidden: true,
    },
  },
  {
    title: intl.get('pages.group.form.basegroup'),
    dataIndex: 'baseName',
    readonly: true,
  },
  ...GroupColumns(intl),
  {
    title: intl.get('pages.group.form.cloneuser'),
    dataIndex: 'clone_user',
    valueType: 'switch',
  },
];

const GroupClone: React.FC<GroupBtnProps> = (props) => {
  const intl = getIntl();
  const handleClone = async (params: ParamsType) => {
    delete params.baseName;
    if (await checkAPI(postAPI('/groups', params))) {
      props.tableRef?.current?.reloadAndRest?.();
      return true;
    }
    return false;
  };

  return (
    <TableOp
      title={intl.get('pages.group.clone.title')}
      trigger={
        <TableBtn
          key="clone"
          icon={PlusOutlined}
          tip={intl.get('app.op.clone')}
        />
      }
      schemaProps={{
        onFinish: handleClone,
        columns: cloneColumns(intl),
        initialValues: props.initialValues,
      }}
      permName="manage.user"
      perm={UserPerm.PermWrite}
      rollback={<PlusOutlined key="clone" />}
      width={500}
    />
  );
};

export default GroupClone;
