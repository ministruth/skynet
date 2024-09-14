import { API_PREFIX } from '@/config';
import { getAPI, getIntl } from '@/utils';
import { LoadingOutlined, ReloadOutlined } from '@ant-design/icons';
import { FormattedMessage } from '@umijs/max';
import { FitAddon } from '@xterm/addon-fit';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { Terminal } from '@xterm/xterm';
import '@xterm/xterm/css/xterm.css';
import { Button } from 'antd';
import moment from 'moment';
import { useEffect, useRef, useState } from 'react';
import { TaskOutputProps } from './output';

const TaskTerminal: React.FC<TaskOutputProps> = (props) => {
  const intl = getIntl();
  const term_ref = useRef(null);
  const term = useRef(new Terminal({ cursorStyle: 'bar' }));
  const fitAddon = useRef(new FitAddon());
  const handleResize = () => {
    fitAddon.current.fit();
  };
  const pos = useRef(0);
  const [polling, setPolling] = useState(true);
  const pid = useRef<NodeJS.Timeout>();
  const [time, setTime] = useState(0);
  const refresh = async () => {
    const msg = await getAPI(`${API_PREFIX}/tasks/${props.id}/output`, {
      pos: pos.current,
    });
    let data = msg.data;
    pos.current = data.pos;
    term.current.write(data.output);
    setTime(Date.now());
  };

  useEffect(() => {
    refresh();
    pid.current = setInterval(refresh, 2000);
    if (term_ref.current) {
      term.current.open(term_ref.current);
      term.current.loadAddon(fitAddon.current);
      term.current.loadAddon(new WebLinksAddon());
    }
    window.addEventListener('resize', handleResize);
    handleResize();
    return () => {
      clearInterval(pid.current);
      window.removeEventListener('resize', handleResize);
    };
  }, []);

  return (
    <>
      <div className="ant-pro-table-list-toolbar">
        <div
          className="ant-pro-table-list-toolbar-container"
          style={{ paddingTop: 0 }}
        >
          <div className="ant-pro-table-list-toolbar-left">
            <div className="ant-pro-table-list-toolbar-title">
              <FormattedMessage
                id="app.table.lastupdate"
                values={{ time: moment(time).format('HH:mm:ss') }}
              />
            </div>
          </div>
          <div className="ant-pro-table-list-toolbar-right">
            <Button
              key="poll"
              type="primary"
              onClick={() => {
                if (polling) {
                  setPolling(false);
                  clearInterval(pid.current);
                  return;
                }
                setPolling(true);
                pid.current = setInterval(refresh, 2000);
              }}
            >
              {polling ? <LoadingOutlined /> : <ReloadOutlined />}
              {polling
                ? intl.get('app.table.polling.stop')
                : intl.get('app.table.polling.start')}
            </Button>
            <div className="ant-pro-table-list-toolbar-setting-items">
              <div className="ant-pro-table-list-toolbar-setting-item">
                <ReloadOutlined onClick={refresh} />
              </div>
            </div>
          </div>
        </div>
      </div>
      <div id="terminal" ref={term_ref} style={{ height: '60vh' }}></div>
    </>
  );
};

export default TaskTerminal;
