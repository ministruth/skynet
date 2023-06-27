import { UserPerm, checkAPI, getIntl, postAPI } from '@/utils';
import { PlusOutlined } from '@ant-design/icons';
import { ParamsType } from '@ant-design/pro-provider';
import { FormattedMessage } from 'react-intl';
import { Columns } from '../layout/table/column';
import TableOp from '../layout/table/opBtn';
import TableBtn from '../layout/table/tableBtn';
import { GroupBtnProps, GroupColumns } from './card';

const cloneColumns: Columns = (intl) => [
  {
    renderFormItem: () => (
      <FormattedMessage id="pages.group.op.clone.content" />
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
    if (await checkAPI(postAPI('/group', params))) {
      props.tableRef?.current?.reloadAndRest?.();
      return true;
    }
    return false;
  };

  return (
    <TableOp
      title={intl.get('pages.group.op.clone.title')}
      trigger={
        <TableBtn
          key="clone"
          icon={PlusOutlined}
          tip={intl.get('pages.group.op.clone.tip')}
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
    />
  );
};

export default GroupClone;
