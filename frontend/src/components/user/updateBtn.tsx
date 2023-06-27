import { checkAPI, getAPI, getIntl, putAPI, UserPerm } from '@/utils';
import { EditOutlined } from '@ant-design/icons';
import { ParamsType } from '@ant-design/pro-provider';
import _ from 'lodash';
import { Columns } from '../layout/table/column';
import TableOp from '../layout/table/opBtn';
import TableBtn from '../layout/table/tableBtn';
import { AvatarColumn, GroupColumn, UserBtnProps } from './card';

const updateColumns: Columns = (intl) => [
  {
    title: intl.get('pages.user.table.username'),
    dataIndex: 'username',
    tooltip: intl.get('pages.user.form.username.tip'),
    fieldProps: {
      maxLength: 32,
    },
    formItemProps: {
      rules: [{ required: true }],
    },
  },
  {
    title: intl.get('pages.user.table.password'),
    dataIndex: 'password',
    fieldProps: {
      placeholder: intl.get('pages.user.form.password.placeholder'),
    },
  },
  AvatarColumn(intl),
  GroupColumn(intl),
];

const UserUpdate: React.FC<UserBtnProps> = (props) => {
  const intl = getIntl();
  const columns = updateColumns(intl);
  const handleUpdate = async (params: ParamsType) => {
    let column: string[] = [];
    _.forEach(params, (v, k) => {
      if (props.initialValues?.[k] != v) column.push(k);
    });
    params['column'] = column;
    if (await checkAPI(putAPI(`/user/${props.initialValues?.id}`, params))) {
      props.tableRef?.current?.reloadAndRest?.();
      return true;
    }
    return false;
  };

  return (
    <TableOp
      trigger={
        <TableBtn
          key="update"
          icon={EditOutlined}
          tip={intl.get('pages.user.op.update.tip')}
        />
      }
      schemaProps={{
        request: async (_params: Record<string, any>, _props: any) => {
          let rsp = await getAPI(`/user/${props.initialValues?.id}/group`);
          props.initialValues!.group = rsp.data.map((e: any) => ({
            value: e.id,
            label: e.name,
          }));
          return props.initialValues!;
        },
        onFinish: handleUpdate,
        columns: columns,
        initialValues: props.initialValues,
      }}
      rollback={<EditOutlined key="update" />}
      permName="manage.user"
      perm={UserPerm.PermWriteExecute}
      width={500}
      title={intl.get('pages.user.op.update.title')}
    />
  );
};

export default UserUpdate;
