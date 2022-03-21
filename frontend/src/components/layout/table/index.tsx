import { getIntl } from '@/utils';
import { LoadingOutlined, ReloadOutlined } from '@ant-design/icons';
import { ParamsType } from '@ant-design/pro-provider';
import ProTable, { ProTableProps } from '@ant-design/pro-table';
import { Button } from 'antd';
import moment from 'moment';
import { useState } from 'react';

type TableProp<T, U extends ParamsType, ValueType = 'text'> = {
  poll?: boolean;
  action?: React.ReactNode[];
  postRequest?: (data: any[]) => any[];
} & ProTableProps<T, U, ValueType>;

function Table<T, U = ParamsType, ValueType = 'text'>(
  props: TableProp<T, U, ValueType>,
) {
  const intl = getIntl();
  const [polling, setPolling] = useState(true);
  const [time, setTime] = useState(() => Date.now());
  const { postRequest, ...rest } = props;

  return (
    <ProTable<T, U, ValueType>
      polling={props.poll ? (polling ? 1000 : undefined) : undefined}
      headerTitle={intl.get('app.table.lastupdate', {
        time: moment(time).format('HH:mm:ss'),
      })}
      postData={(data: any[]) => {
        setTime(Date.now());
        if (postRequest) return postRequest(data);
        return data;
      }}
      toolbar={{
        actions: [
          props.poll
            ? [
                props.action,
                <Button
                  key="poll"
                  type="primary"
                  onClick={() => {
                    if (polling) {
                      setPolling(false);
                      return;
                    }
                    setPolling(true);
                  }}
                >
                  {polling ? <LoadingOutlined /> : <ReloadOutlined />}
                  {polling
                    ? intl.get('app.table.polling.stop')
                    : intl.get('app.table.polling.start')}
                </Button>,
              ]
            : props.action,
        ],
      }}
      cardBordered
      bordered
      search={{
        defaultCollapsed: false,
        collapseRender: false,
      }}
      pagination={{
        defaultPageSize: 10,
        pageSizeOptions: ['5', '10', '20', '50'],
        showQuickJumper: true,
      }}
      options={{
        density: false,
      }}
      {...rest}
    />
  );
}

export default Table;
