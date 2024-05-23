import {
  checkAPI,
  getAPI,
  getIntl,
  putAPI,
  StringIntl,
  UserPerm,
} from '@/utils';
import { EditOutlined } from '@ant-design/icons';
import { ProFormColumnsType } from '@ant-design/pro-components';
import { ParamsType } from '@ant-design/pro-provider';
import _ from 'lodash';
import TableOp from '../layout/table/opBtn';
import TableBtn from '../layout/table/tableBtn';
import { AvatarColumn, GroupColumn, UserBtnProps } from './card';

const updateColumns: (
  intl: StringIntl,
  root: boolean,
) => ProFormColumnsType[] = (intl, root) => {
  let ret = [
    {
      title: intl.get('pages.user.table.username'),
      dataIndex: 'username',
      tooltip: intl.get('pages.user.form.username.tip'),
      fieldProps: {
        maxLength: 32,
        disabled: root,
      },
      formItemProps: {
        rules: [{ required: true }],
      },
    },
    {
      title: intl.get('pages.user.table.password'),
      dataIndex: 'password',
      valueType: 'password',
      fieldProps: {
        placeholder: intl.get('pages.user.form.password.placeholder'),
      },
    },
    AvatarColumn(intl),
  ];
  if (!root) {
    ret.push(GroupColumn(intl));
  }
  return ret as any;
};

const UserUpdate: React.FC<UserBtnProps> = (props) => {
  const intl = getIntl();
  const columns = updateColumns(
    intl,
    props.initialValues?.id === '00000000-0000-0000-0000-000000000000',
  );
  const handleUpdate = async (params: ParamsType) => {
    _.forEach(params, (v, k) => {
      if (_.isEqual(props.initialValues?.[k], v)) delete params[k];
    });
    params['group'] = params['group']?.map((x: { value: string }) => x.value);

    if (await checkAPI(putAPI(`/users/${props.initialValues?.id}`, params))) {
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
          tip={intl.get('app.op.update')}
        />
      }
      schemaProps={{
        request: async (_params: Record<string, any>, _props: any) => {
          let rsp = await getAPI(`/users/${props.initialValues?.id}/groups`);
          props.initialValues!.group = rsp.data.map((e: any) => ({
            value: e.id,
            label: e.name,
            key: e.id, // labelInValue has these additinal object
            title: e.name, // labelInValue has these additinal object
          }));
          props.initialValues!.password = '';
          return props.initialValues!;
        },
        onFinish: handleUpdate,
        columns: columns as any,
        initialValues: props.initialValues,
      }}
      rollback={<EditOutlined key="update" />}
      permName="manage.user"
      perm={UserPerm.PermWrite}
      disabled={props.disabled}
      width={500}
      title={intl.get('pages.user.op.update.title')}
      changedSubmit={true}
    />
  );
};

export default UserUpdate;
