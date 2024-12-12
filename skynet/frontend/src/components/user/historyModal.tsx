import { getAPI, getIntl, paramSort, paramTime } from '@/utils';
import { ParamsType } from '@ant-design/pro-components';
import { ProColumns } from '@ant-design/pro-table';
import { Modal } from 'antd';
import { SortOrder } from 'antd/es/table/interface';
import GeoIP from '../geoip';
import Table from '../layout/table';
import { IDColumn } from '../layout/table/column';
import styles from './style.less';

export interface HistoryModalProps {
  uid: string;
  open: boolean;
  setOpen: (status: boolean) => void;
}

const HistoryModal: React.FC<HistoryModalProps> = (props) => {
  const intl = getIntl();

  let columns: ProColumns[] = [
    IDColumn(intl),
    {
      title: intl.get('tables.ip'),
      dataIndex: 'ip',
      align: 'center',
      render: (_, row) => <GeoIP value={row.ip} />,
    },
    {
      title: intl.get('tables.time'),
      dataIndex: 'time',
      align: 'center',
      valueType: 'dateTime',
      sorter: true,
      hideInSearch: true,
    },
    {
      title: intl.get('tables.time'),
      dataIndex: 'time',
      valueType: 'dateRange',
      hideInTable: true,
      search: {
        transform: (value) => {
          return {
            timeStart: value[0],
            timeEnd: value[1],
          };
        },
      },
    },
  ];
  const request = async (
    params?: ParamsType,
    sort?: Record<string, SortOrder>,
  ) => {
    const msg = await getAPI(`/users/${props.uid}/histories`, {
      ip: params?.ip,
      time_sort: paramSort(sort?.time) || 'desc',
      time_start: paramTime(params?.timeStart),
      time_end: paramTime(params?.timeEnd, true),
      page: params?.current,
      size: params?.pageSize,
    });
    return {
      data: msg.data.data,
      success: true,
      total: msg.data.total,
    };
  };

  return (
    <Modal
      title={intl.get('pages.history.title')}
      open={props.open}
      footer={null}
      onCancel={() => {
        props.setOpen(false);
      }}
      width={700}
      destroyOnClose={true}
      maskClosable={false}
    >
      <Table
        rowKey="id"
        request={request}
        columns={columns}
        cardBordered={{ search: true, table: false }}
        search={{
          defaultCollapsed: false,
          collapseRender: false,
          span: 12,
          labelWidth: 50,
          className: styles['searchbox'],
        }}
        cardProps={{ bodyStyle: { paddingInline: 0, paddingBlock: 0 } }}
      />
    </Modal>
  );
};

export default HistoryModal;
