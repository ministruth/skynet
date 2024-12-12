import { checkAPI, getIntl, putAPI, StringIntl, UserPerm } from '@/utils';
import { DisconnectOutlined, LinkOutlined } from '@ant-design/icons';
import { ActionType } from '@ant-design/pro-table';
import confirm from '../layout/modal';
import TableBtn from '../layout/table/tableBtn';

export interface PluginAbleProps {
  tableRef: React.MutableRefObject<ActionType | undefined>;
  enable: boolean;
  pid: string;
  pname: string;
}

const handleAble = (
  intl: StringIntl,
  ref: React.MutableRefObject<ActionType | undefined>,
  id: string,
  name: string,
  enable: boolean,
) => {
  if (enable) {
    return checkAPI(putAPI(`/plugins/${id}`, { enable: enable })).then(() =>
      ref.current?.reloadAndRest?.(),
    );
  } else {
    confirm({
      title: intl.get('pages.plugin.disable.title', {
        name: name,
      }),
      content: intl.get('pages.plugin.disable.content'),
      onOk() {
        return checkAPI(putAPI(`/plugins/${id}`, { enable: enable })).then(() =>
          ref.current?.reloadAndRest?.(),
        );
      },
      intl: intl,
    });
  }
};

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
          handleAble(
            intl,
            props.tableRef,
            props.pid,
            props.pname,
            !props.enable,
          )
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
          handleAble(
            intl,
            props.tableRef,
            props.pid,
            props.pname,
            !props.enable,
          )
        }
      />
    );
};

export default PluginAble;
