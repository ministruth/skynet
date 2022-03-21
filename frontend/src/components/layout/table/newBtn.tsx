import { PlusOutlined } from '@ant-design/icons';
import { FormSchema } from '@ant-design/pro-form/lib/components/SchemaForm';
import { Button } from 'antd';
import { FormattedMessage } from 'react-intl';
import TableOp, { TableOpProps } from './opBtn';

const TableNew: React.FC<TableOpProps & FormSchema> = (props) => {
  return (
    <TableOp
      trigger={
        <Button key="add" type="primary">
          <PlusOutlined style={{ marginRight: '8px' }} />
          <FormattedMessage id="app.table.add" />
        </Button>
      }
      rollback={
        <Button key="add" type="primary" disabled>
          <PlusOutlined style={{ marginRight: '8px' }} />
          <FormattedMessage id="app.table.add" />
        </Button>
      }
      {...props}
    />
  );
};
export default TableNew;
