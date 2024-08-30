import { PLUGIN_ID } from '@/config';
import { request } from '@umijs/max';
import { FitAddon } from '@xterm/addon-fit';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { Terminal } from '@xterm/xterm';
import '@xterm/xterm/css/xterm.css';
import Cookies from 'js-cookie';
import { useEffect, useRef } from 'react';
import { v4 as uuidv4 } from 'uuid';
import { TabItemProps } from './default';
import { FrontendMessage } from './msg';
import ShellSelect from './shellSelect';

export interface ShellTabProps {
  id: string;
  name: string;
  ip: string;
}

const ShellTab: React.FC<TabItemProps & ShellTabProps> = (props) => {
  const ref = useRef(null);
  const term = useRef(new Terminal({ cursorBlink: true }));
  const fitAddon = useRef(new FitAddon());
  const token = useRef('');
  const shellSize = useRef({
    rows: 0,
    cols: 0,
  });
  const handleResize = () => {
    fitAddon.current.fit();
  };

  const ws = useRef<WebSocket | null>(null);
  const url =
    ('https:' == document.location.protocol ? 'wss://' : 'ws://') +
    (process.env.NODE_ENV === 'production'
      ? window.location.host
      : 'localhost:8001') +
    `/api/plugins/${PLUGIN_ID}/ws`;
  const connect = (cmd: string) => {
    if (ws.current) {
      ws.current.onmessage = null;
      ws.current.onerror = null;
      ws.current.onclose = null;
      ws.current.close();
      term.current.writeln('');
    }
    term.current.writeln(`Connecting to ${props.name}(${props.ip})...`);
    request('/token', {
      method: 'get',
    }).then((rsp) => {
      if (rsp) {
        ws.current = new WebSocket(
          url + '?X-CSRF-Token=' + Cookies.get('CSRF_TOKEN'),
        );
        ws.current.binaryType = 'arraybuffer';
        ws.current.onopen = (e) => {
          token.current = uuidv4();
          let msg: FrontendMessage = {
            id: props.id,
            data: {
              oneofKind: 'shellConnect',
              shellConnect: {
                token: token.current,
                cmd: cmd,
                rows: shellSize.current.rows,
                cols: shellSize.current.cols,
              },
            },
          };
          ws.current?.send(FrontendMessage.toBinary(msg));
        };
        ws.current.onmessage = (e) => {
          let data = FrontendMessage.fromBinary(new Uint8Array(e.data)).data;
          switch (data.oneofKind) {
            case 'shellOutput':
              term.current.write(data.shellOutput.data);
              break;
            case 'shellError':
              term.current.writeln('Error: ' + data.shellError.error);
              break;
            default:
              console.log('Unknown message: ', data);
              break;
          }
        };
        ws.current.onclose = (e) => {
          term.current.writeln('\r\nConnection closed.');
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
      term.current.loadAddon(fitAddon.current);
      term.current.loadAddon(new WebLinksAddon());
      term.current.onResize((e) => {
        shellSize.current.rows = e.rows;
        shellSize.current.cols = e.cols;
        let msg: FrontendMessage = {
          data: {
            oneofKind: 'shellResize',
            shellResize: {
              token: token.current,
              rows: e.rows,
              cols: e.cols,
            },
          },
        };
        ws.current?.send(FrontendMessage.toBinary(msg));
      });
      term.current.onData((e) => {
        let msg: FrontendMessage = {
          data: {
            oneofKind: 'shellInput',
            shellInput: {
              token: token.current,
              data: Buffer.from(e),
            },
          },
        };
        ws.current?.send(FrontendMessage.toBinary(msg));
      });
    }
    window.addEventListener('resize', handleResize);
    handleResize();
    return () => {
      ws.current?.close();
      window.removeEventListener('resize', handleResize);
    };
  }, []);
  return (
    <>
      <ShellSelect onClick={connect} />
      <div id="terminal" ref={ref} style={{ height: '60vh' }}></div>
    </>
  );
};

export default ShellTab;
