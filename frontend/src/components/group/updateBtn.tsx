import { checkAPI, getIntl, putAPI, UserPerm } from '@/utils';
import { EditOutlined } from '@ant-design/icons';
import { ParamsType } from '@ant-design/pro-provider';
import _ from 'lodash';
import TableOp from '../layout/table/opBtn';
import TableBtn from '../layout/table/tableBtn';
import { GroupBtnProps, GroupColumns } from './card';

const GroupUpdate: React.FC<GroupBtnProps> = (props) => {
  const intl = getIntl();
  const updateColumns = _.cloneDeep(GroupColumns(intl));
  updateColumns[0].readonly = props.initialValues?.name === 'root';
  const handleUpdate = async (params: ParamsType) => {
    let column: string[] = [];
    _.forEach(params, (v, k) => {
      if (props.initialValues?.[k] != v) column.push(k);
    });
    params['column'] = column;
    if (await checkAPI(putAPI(`/group/${props.initialValues?.id}`, params))) {
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
          tip={intl.get('pages.group.op.update.tip')}
        />
      }
      rollback={<EditOutlined key="update" />}
      permName="manage.user"
      perm={UserPerm.PermWriteExecute}
      schemaProps={{
        onFinish: handleUpdate,
        columns: updateColumns,
        initialValues: props.initialValues,
      }}
      width={500}
    />
  );
};

export default GroupUpdate;
