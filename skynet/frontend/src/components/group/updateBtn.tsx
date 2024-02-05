import { checkAPI, getIntl, putAPI, UserPerm } from '@/utils';
import { EditOutlined } from '@ant-design/icons';
import { ParamsType } from '@ant-design/pro-provider';
import _ from 'lodash';
import TableOp from '../layout/table/opBtn';
import TableBtn from '../layout/table/tableBtn';
import { GroupBtnProps, GroupColumns } from './card';

const GroupUpdate: React.FC<GroupBtnProps> = (props) => {
  const intl = getIntl();
  const handleUpdate = async (params: ParamsType) => {
    _.forEach(params, (v, k) => {
      if (_.isEqual(props.initialValues?.[k], v)) delete params[k];
    });
    if (await checkAPI(putAPI(`/groups/${props.initialValues?.id}`, params))) {
      props.tableRef?.current?.reloadAndRest?.();
      return true;
    }
    return false;
  };

  return (
    <TableOp
      title={intl.get('pages.group.op.update.title')}
      trigger={
        <TableBtn
          key="update"
          icon={EditOutlined}
          tip={intl.get('app.op.update.tip')}
        />
      }
      rollback={<EditOutlined key="update" />}
      permName="manage.user"
      perm={UserPerm.PermWrite}
      schemaProps={{
        onFinish: handleUpdate,
        columns: GroupColumns(intl),
        initialValues: props.initialValues,
      }}
      width={500}
      changedSubmit={true}
    />
  );
};

export default GroupUpdate;
