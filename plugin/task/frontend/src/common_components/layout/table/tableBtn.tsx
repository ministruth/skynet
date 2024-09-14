import { checkPerm, UserPerm } from '@/utils';
import { useModel } from '@umijs/max';
import { Tooltip } from 'antd';

export interface TableBtnProps {
  icon: React.ElementType;
  tip: string;
  onClick?: (event: any) => void;
  color?: string;
  disabled?: boolean;
  perm?: UserPerm;
  permName?: string;
}

const TableBtn: React.FC<TableBtnProps> = (props) => {
  const { access } = useModel('@@qiankunStateFromMaster');
  const disabled =
    props.disabled ||
    (props.perm &&
      props.permName &&
      !checkPerm(access, props.permName, props.perm));
  return (
    <Tooltip open={disabled ? false : undefined} title={props.tip}>
      <props.icon
        style={
          disabled
            ? undefined
            : { color: props.color ? props.color : '#1890ff' }
        }
        onClick={disabled ? undefined : props.onClick}
      />
    </Tooltip>
  );
};
export default TableBtn;
