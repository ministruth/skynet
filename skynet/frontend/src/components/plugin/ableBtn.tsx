import { checkAPI, getIntl, putAPI, UserPerm } from '@/utils';
import { DisconnectOutlined, LinkOutlined } from '@ant-design/icons';
import { ActionType } from '@ant-design/pro-table';
import TableBtn from '../layout/table/tableBtn';

export interface PluginAbleProps {
  tableRef: React.MutableRefObject<ActionType | undefined>;
  enable: boolean;
  pid: string;
  pname: string;
}

const PluginAble: React.FC<PluginAbleProps> = (props) => {
  const intl = getIntl();
  if (props.enable)
    return (
      <TableBtn
        icon={DisconnectOutlined}
        tip={intl.get('pages.plugin.disable.tip')}
        perm={UserPerm.PermWrite}
        permName="manage.plugin"
        onClick={() =>
          checkAPI(
            putAPI(`/plugins/${props.pid}`, { enable: !props.enable }),
          ).then(() => props.tableRef.current?.reloadAndRest?.())
        }
      />
    );
  else
    return (
      <TableBtn
        icon={LinkOutlined}
        tip={intl.get('pages.plugin.enable.tip')}
        perm={UserPerm.PermWrite}
        permName="manage.plugin"
        onClick={() =>
          checkAPI(
            putAPI(`/plugins/${props.pid}`, { enable: !props.enable }),
          ).then(() => props.tableRef.current?.reloadAndRest?.())
        }
      />
    );
};

export default PluginAble;
