import { StringIntl } from '@/utils';
import { ProFormColumnsType } from '@ant-design/pro-form';
import { ProColumns } from '@ant-design/pro-table';

export type Columns = (intl: StringIntl) => ProFormColumnsType[];
export type Column = (intl: StringIntl) => ProFormColumnsType;

export const IDColumn: (intl: StringIntl) => ProColumns = (intl) => {
  return {
    title: intl.get('app.table.id'),
    ellipsis: true,
    dataIndex: 'id',
    align: 'center',
    copyable: true,
    hideInSearch: true,
    onCell: () => {
      return {
        style: {
          minWidth: 150,
          maxWidth: 150,
        },
      };
    },
  };
};

export const SearchColumn: (intl: StringIntl) => ProColumns = (intl) => {
  return {
    title: intl.get('app.table.searchtext'),
    key: 'text',
    hideInTable: true,
  };
};

export const CreatedAtColumn: (intl: StringIntl) => ProColumns[] = (intl) => [
  {
    title: intl.get('app.table.createdat'),
    dataIndex: 'created_at',
    align: 'center',
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
          created_start: value[0],
          created_end: value[1],
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
          updated_start: value[0],
          updated_end: value[1],
        };
      },
    },
  },
];
