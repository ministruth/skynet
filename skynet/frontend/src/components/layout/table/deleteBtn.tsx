import { deleleAPI, getIntl, StringIntl, UserPerm } from '@/utils';
import { DeleteOutlined } from '@ant-design/icons';
import { ActionType } from '@ant-design/pro-table';
import confirm from '../modal';
import TableBtn from './tableBtn';

export interface TableDeleteProps {
  disabled?: boolean;
  perm?: UserPerm;
  permName?: string;
  tableRef: React.MutableRefObject<ActionType | undefined>;
  url: string;
  refresh?: boolean;
  confirmTitle: string;
  confirmContent?: string;
  confirmData?: {};
}

const handleDelete = async (
  intl: StringIntl,
  ref: React.MutableRefObject<ActionType | undefined>,
  url: string,
  refresh: boolean,
  title: string,
  content?: string,
  data?: any,
) => {
  confirm({
    title: title,
    content: content ? content : intl.get('app.confirm'),
    onOk() {
      return new Promise((resolve, reject) => {
        deleleAPI(url, data ? data : {}).then((rsp) => {
          if (rsp && rsp.code === 0) {
            ref.current?.reloadAndRest?.();
            if (refresh) window.location.reload();
            resolve(rsp);
          } else {
            reject(rsp);
          }
        });
      });
    },
    intl: intl,
  });
};

const TableDelete: React.FC<TableDeleteProps> = (props) => {
  const intl = getIntl();
  return (
    <TableBtn
      icon={DeleteOutlined}
      tip={intl.get('app.op.delete.tip')}
      color="#ff4d4f"
      disabled={props.disabled}
      perm={props.perm}
      permName={props.permName}
      onClick={() =>
        handleDelete(
          intl,
          props.tableRef,
          props.url,
          props.refresh ?? false,
          props.confirmTitle,
          props.confirmContent,
          props.confirmData,
        )
      }
    />
  );
};
export default TableDelete;
