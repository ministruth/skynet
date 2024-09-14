import { PlusOutlined } from '@ant-design/icons';
import { Button } from 'antd';
import { FormattedMessage } from 'react-intl';
import { ModalSchemaProps } from '../modalSchema';
import TableOp, { TableOpProps } from './opBtn';

const TableNew: React.FC<TableOpProps & ModalSchemaProps> = (props) => {
  return (
    <TableOp
      trigger={
        <Button key="add" type="primary">
          <PlusOutlined />
          <FormattedMessage id="app.op.add" />
        </Button>
      }
      rollback={
        <Button key="add" type="primary" disabled>
          <PlusOutlined />
          <FormattedMessage id="app.op.add" />
        </Button>
      }
      {...props}
    />
  );
};
export default TableNew;
