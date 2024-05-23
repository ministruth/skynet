import RowItem from '@/common_components/layout/rowItem';
import { API_PREFIX } from '@/config';
import { getAPI, getIntl } from '@/utils';
import { FormattedMessage } from '@umijs/max';
import { Button, Popconfirm, Select } from 'antd';
import React, { useEffect, useState } from 'react';

const request = async () => {
  return (await getAPI(`${API_PREFIX}/settings/shell`)).data.map((v: any) => ({
    value: v,
    label: v,
  }));
};

enum Status {
  Pending,
  Invalid,
  Ready,
  Working,
}

export interface ShellSelectProps {
  onClick: (cmd: string) => void;
}

const ShellSelect: React.FC<ShellSelectProps> = (props) => {
  const intl = getIntl();
  const [options, setOptions] = useState([]);
  const [value, setValue] = useState<string | undefined>(undefined);
  const [status, setStatus] = useState(Status.Pending);

  useEffect(() => {
    request().then((rsp) => {
      if (rsp) {
        setOptions(rsp);
        if (rsp.length > 0) {
          setValue(rsp[0].value);
          setStatus(Status.Ready);
        } else {
          setStatus(Status.Invalid);
        }
      }
    });
  }, []);
  let btn = (
    <Button
      type={status === Status.Working ? 'default' : 'primary'}
      danger={status === Status.Working}
      disabled={status === Status.Pending || status === Status.Invalid}
    >
      <FormattedMessage
        id={
          status === Status.Working
            ? 'pages.view.card.reconnect.text'
            : 'pages.view.card.connect.text'
        }
      />
    </Button>
  );
  let connectBtn = React.cloneElement(btn, {
    onClick: () => {
      props.onClick(value ?? '');
      setStatus(Status.Working);
    },
  });
  let reconnectBtn = (
    <Popconfirm
      title={intl.get('pages.view.card.reconnect.title')}
      description={intl.get('pages.view.card.reconnect.content')}
      onConfirm={() => props.onClick(value ?? '')}
    >
      {btn}
    </Popconfirm>
  );
  return (
    <RowItem
      span={{ xs: 6, md: 2 }}
      text={<FormattedMessage id="pages.view.card.shell.text" />}
      item={
        <>
          <Select
            options={options}
            value={value}
            onChange={(v) => setValue(v)}
            loading={status === Status.Pending}
            style={{ width: '190px', marginRight: '10px' }}
            placeholder={intl.get('pages.view.card.shell.placeholder')}
          />
          {status === Status.Working ? reconnectBtn : connectBtn}
        </>
      }
    ></RowItem>
  );
};

export default ShellSelect;
