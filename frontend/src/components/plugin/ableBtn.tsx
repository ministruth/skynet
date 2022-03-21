import { checkAPI, getIntl, ping, putAPI, StringIntl } from '@/utils';
import { DisconnectOutlined, LinkOutlined } from '@ant-design/icons';
import { ActionType } from '@ant-design/pro-table';
import { history, useModel } from 'umi';
import confirm from '../layout/modal';
import TableBtn from '../layout/table/tipBtn';

export interface PluginAbleProps {
  tableRef: React.MutableRefObject<ActionType | undefined>;
  enable: boolean;
  pluginID: string;
}

const handleAble = (
  intl: StringIntl,
  id: string,
  enable: boolean,
  refresh: () => Promise<void>,
) => {
  let t: NodeJS.Timer;
  if (enable)
    return checkAPI(putAPI(`/plugin/${id}`, { enable: enable })).then((rsp) => {
      if (rsp) window.location.reload();
    });
  confirm({
    title: 'pages.plugin.table.disable.title',
    content: 'pages.plugin.table.disable.content',
    onOk() {
      return new Promise((resolve, reject) => {
        putAPI(`/plugin/${id}`, { enable: enable }).then(async (rsp) => {
          if (rsp && rsp.code === 0)
            t = setInterval(async () => {
              if (await ping()) {
                clearInterval(t);
                resolve(rsp);
                refresh().then(() => {
                  history.push('/');
                });
              }
            }, 1000);
          else reject(rsp);
        });
      });
    },
    intl: intl,
  });
};

const PluginAble: React.FC<PluginAbleProps> = (props) => {
  const intl = getIntl();
  const { refresh } = useModel('@@initialState');
  if (props.enable)
    return (
      <TableBtn
        icon={DisconnectOutlined}
        tip={intl.get('pages.plugin.table.disabletip')}
        onClick={() => handleAble(intl, props.pluginID, !props.enable, refresh)}
      />
    );
  else
    return (
      <TableBtn
        icon={LinkOutlined}
        tip={intl.get('pages.plugin.table.enabletip')}
        onClick={() => handleAble(intl, props.pluginID, !props.enable, refresh)}
      />
    );
};

export default PluginAble;
