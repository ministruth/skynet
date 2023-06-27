import { checkPerm, getIntl, UserPerm } from '@/utils';
import { useAccess, useModel } from '@umijs/max';
import { ReactElement } from 'react';
import ModalSchema, { ModalSchemaProps } from './modalSchema';

export interface TableOpProps {
  disabled?: boolean;
  perm?: UserPerm;
  permName?: string;
  rollback?: ReactElement<any, any>;
}

const TableOp: React.FC<TableOpProps & ModalSchemaProps> = (props) => {
  const intl = getIntl();
  const { initialState } = useModel('@@initialState');
  const access = useAccess();

  if (
    props.disabled ||
    (props.perm &&
      props.permName &&
      !checkPerm(initialState?.signin, access, props.permName, props.perm))
  )
    return props.rollback ? props.rollback : <></>;
  else
    return (
      <ModalSchema
        title={props.title}
        trigger={props.trigger}
        width={props.width}
        schemaProps={{
          layout: 'horizontal',
          autoFocusFirstInput: true,
          labelCol: { span: 6 },
          ...props.schemaProps,
        }}
      />
    );
};
export default TableOp;
