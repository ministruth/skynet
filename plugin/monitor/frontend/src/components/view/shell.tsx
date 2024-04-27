import { PLUGIN_ID } from '@/config';
import { getIntl } from '@/utils';
import { ReloadOutlined } from '@ant-design/icons';
import { request } from '@umijs/max';
import { FitAddon } from '@xterm/addon-fit';
import { Terminal } from '@xterm/xterm';
import '@xterm/xterm/css/xterm.css';
import { Button, Popconfirm } from 'antd';
import Cookies from 'js-cookie';
import { useEffect, useRef, useState } from 'react';
import { TabItemProps } from './default';

export interface ShellTabProps {
  id: string;
  name: string;
  ip: string;
}

const ShellTab: React.FC<TabItemProps & ShellTabProps> = (props) => {
  const intl = getIntl();
  const ref = useRef(null);
  const term = useRef(new Terminal({ cursorBlink: true }));
  const fitAddon = new FitAddon();
  const [windowSize, setWindowSize] = useState({
    width: 0,
    height: 0,
  });
  const handleResize = () => {
    setWindowSize({
      width: window.innerWidth,
      height: window.innerHeight,
    });
    fitAddon.fit();
  };

  const ws = useRef<WebSocket | null>(null);
  const url =
    ('https:' == document.location.protocol ? 'wss://' : 'ws://') +
    (process.env.NODE_ENV === 'production'
      ? window.location.host
      : 'localhost:8001') +
    `/api/plugins/${PLUGIN_ID}/ws`;
  const connect = () => {
    term.current.writeln(`Connecting to ${props.name}(${props.ip})...`);
    request('/token', {
      method: 'get',
    }).then((rsp) => {
      if (rsp) {
        ws.current = new WebSocket(
          url + '?X-CSRF-Token=' + Cookies.get('CSRF_TOKEN'),
        );
        ws.current.onopen = (e) => {
          console.log('open', e);
        };
        ws.current.onmessage = (e) => {
          console.log(e);
        };
        ws.current.onclose = (e) => {
          term.current.writeln('Connection closed.');
        };
        ws.current.onerror = (e) => {
          term.current.writeln('Error.');
        };
      }
    });
  };

  useEffect(() => {
    if (ref.current) {
      term.current.open(ref.current);
      term.current.loadAddon(fitAddon);
      term.current.onResize(function (evt) {
        console.log(evt);
      });
    }
    window.addEventListener('resize', handleResize);
    handleResize();
    connect();
    return () => window.removeEventListener('resize', handleResize);
  }, []);
  return (
    <div id="terminal" ref={ref} style={{ height: '60vh' }}>
      <Popconfirm
        title={intl.get('pages.view.card.shell.retry.title')}
        description={intl.get('pages.view.card.shell.retry.content')}
        onConfirm={() => {
          if (ws.current) {
            ws.current.onclose = null;
            ws.current.close();
          }
          connect();
        }}
      >
        <Button
          size="small"
          style={{
            position: 'absolute',
            zIndex: 100,
            right: '45px',
            top: '20px',
          }}
          icon={<ReloadOutlined />}
          danger
          ghost
        />
      </Popconfirm>
    </div>
  );
};

export default ShellTab;
