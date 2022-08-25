import { checkAPI, getIntl, putAPI, UserPerm } from '@/utils';
import { EditOutlined } from '@ant-design/icons';
import { ParamsType } from '@ant-design/pro-provider';
import _ from 'lodash';
import TableOp from '../layout/table/opBtn';
import TableBtn from '../layout/table/tipBtn';
import { GroupBtnProps, GroupColumns } from './card';

const handleUpdate = (id: string, params: ParamsType) =>
  checkAPI(putAPI(`/group/${id}`, params));

const GroupUpdate: React.FC<GroupBtnProps> = (props) => {
  const intl = getIntl();
  const updateColumns = _.cloneDeep(GroupColumns(intl));
  updateColumns[0].readonly = props.initialValues?.name === 'root';
  return (
    <TableOp
      trigger={
        <TableBtn
          key="update"
          icon={EditOutlined}
          tip={intl.get('pages.group.table.updatetip')}
        />
      }
      rollback={<EditOutlined key="update" />}
      permName="manage.group"
      perm={UserPerm.PermWriteExecute}
      finish={(values) => handleUpdate(props.initialValues?.id, values)}
      width={500}
      title={intl.get('pages.group.table.update')}
      columns={updateColumns}
      {...props}
    />
  );
};

export default GroupUpdate;
