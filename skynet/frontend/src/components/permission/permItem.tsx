import { UserPerm } from '@/utils';
import { Checkbox } from 'antd';
import { CheckboxChangeEvent } from 'antd/es/checkbox';
import styles from './style.less';

export interface PermItemProps {
  basePerm: UserPerm;
  perm: UserPerm;
  onChange?: (perm: UserPerm) => void;
  disabled?: boolean;
}

const PermItem: React.FC<PermItemProps> = (props) => {
  let checkR = false,
    checkW = false;
  let baseR = false,
    baseW = false;
  if (props.perm != UserPerm.PermBan) {
    checkR = (props.perm & UserPerm.PermRead) > 0;
    checkW = (props.perm & UserPerm.PermWrite) > 0;
  }
  if (props.basePerm != UserPerm.PermBan) {
    baseR = (props.basePerm & UserPerm.PermRead) > 0;
    baseW = (props.basePerm & UserPerm.PermWrite) > 0;
  }
  let sum = +checkR + +checkW;
  let baseSum = +baseR + +baseW;

  const change = (checked: boolean, ok: UserPerm, cancel: UserPerm) => {
    if (props.onChange) props.onChange(checked ? ok : cancel);
  };

  const banChanged =
    (props.perm === UserPerm.PermBan && props.basePerm !== UserPerm.PermBan) ||
    (props.perm !== UserPerm.PermBan && props.basePerm === UserPerm.PermBan);
  const changedStyle = { color: 'orange' };

  return (
    <>
      <Checkbox
        indeterminate={sum > 0 && sum < 2}
        checked={sum === 2}
        onChange={(e: CheckboxChangeEvent) => {
          change(e.target.checked, UserPerm.PermAll, UserPerm.PermNone);
        }}
        style={
          !banChanged &&
          (sum === baseSum ||
            (sum > 0 && sum < 2 && baseSum > 0 && baseSum < 2))
            ? {}
            : changedStyle
        }
        disabled={props.disabled || props.perm === UserPerm.PermBan}
      >
        A
      </Checkbox>
      <Checkbox
        className={styles['perm-item']}
        checked={checkR}
        onChange={(e: CheckboxChangeEvent) =>
          change(
            e.target.checked,
            props.perm | UserPerm.PermRead,
            props.perm & ~UserPerm.PermRead,
          )
        }
        style={checkR !== baseR || banChanged ? changedStyle : {}}
        disabled={props.disabled || props.perm === UserPerm.PermBan}
      >
        R
      </Checkbox>
      <Checkbox
        className={styles['perm-item']}
        checked={checkW}
        onChange={(e: CheckboxChangeEvent) =>
          change(
            e.target.checked,
            props.perm | UserPerm.PermWrite,
            props.perm & ~UserPerm.PermWrite,
          )
        }
        style={checkW !== baseW || banChanged ? changedStyle : {}}
        disabled={props.disabled || props.perm === UserPerm.PermBan}
      >
        W
      </Checkbox>
    </>
  );
};

export default PermItem;
