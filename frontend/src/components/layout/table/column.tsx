import { StringIntl } from '@/utils';
import { ProFormColumnsType } from '@ant-design/pro-form';
import { ProColumns } from '@ant-design/pro-table';

export type Columns = (intl: StringIntl) => ProFormColumnsType[];

export const CreatedAtColumn: (intl: StringIntl) => ProColumns[] = (intl) => [
  {
    title: intl.get('app.table.createdat'),
    dataIndex: 'created_at',
    align: 'center',
    width: 180,
    valueType: 'dateTime',
    sorter: true,
    hideInSearch: true,
  },
  {
    title: intl.get('app.table.createdat'),
    dataIndex: 'created_at',
    valueType: 'dateRange',
    hideInTable: true,
    search: {
      transform: (value) => {
        return {
          createdStart: value[0],
          createdEnd: value[1],
        };
      },
    },
  },
];

export const UpdatedAtColumn: (intl: StringIntl) => ProColumns[] = (intl) => [
  {
    title: intl.get('app.table.updatedat'),
    dataIndex: 'updated_at',
    align: 'center',
    width: 180,
    valueType: 'dateTime',
    sorter: true,
    hideInSearch: true,
  },
  {
    title: intl.get('app.table.updatedat'),
    dataIndex: 'updated_at',
    valueType: 'dateRange',
    hideInTable: true,
    search: {
      transform: (value) => {
        return {
          updatedStart: value[0],
          updatedEnd: value[1],
        };
      },
    },
  },
];
