import { PlusOutlined } from "@ant-design/icons";
import { Button } from "antd";
import { FormattedMessage } from "react-intl";
import { ModalSchemaProps } from "../modalSchema";
import TableOp, { TableOpProps } from "./opBtn";

const TableNew: React.FC<TableOpProps & ModalSchemaProps> = (props) => {
  return (
    <TableOp
      trigger={
        <Button key="add" type="primary">
          <PlusOutlined style={{ marginRight: "8px" }} />
          <FormattedMessage id="app.op.add" />
        </Button>
      }
      rollback={
        <Button key="add" type="primary" disabled>
          <PlusOutlined style={{ marginRight: "8px" }} />
          <FormattedMessage id="app.op.add" />
        </Button>
      }
      {...props}
    />
  );
};
export default TableNew;
