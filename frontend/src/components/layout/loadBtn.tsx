import { Button, ButtonProps } from 'antd';
import { useState } from 'react';

export interface LoadBtnProps extends Omit<ButtonProps, 'onClick'> {
  onClick?: (
    e: React.MouseEvent<HTMLAnchorElement> &
      React.MouseEvent<HTMLButtonElement>,
  ) => Promise<any>;
}

const LoadBtn: React.FC<LoadBtnProps> = (props) => {
  const [loadings, setLoadings] = useState<boolean>(false);
  const { onClick, ...rest } = props;
  const click = onClick
    ? (
        e: React.MouseEvent<HTMLAnchorElement> &
          React.MouseEvent<HTMLButtonElement>,
      ) => {
        setLoadings(true);
        onClick(e).finally(() => setLoadings(false));
      }
    : undefined;
  return (
    <Button {...rest} loading={loadings} onClick={click}>
      {props.children}
    </Button>
  );
};

export default LoadBtn;
