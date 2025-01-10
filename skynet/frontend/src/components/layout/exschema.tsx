import { BetaSchemaForm } from '@ant-design/pro-components';
import { FormSchema } from '@ant-design/pro-components/node_modules/@ant-design/pro-form/es/components/SchemaForm';
import { isEqual } from 'lodash';
import { forwardRef, useImperativeHandle, useRef, useState } from 'react';

export interface ExSchemaProps {
  onSubmit: (
    param: Record<string, any>,
    initial: Record<string, any>,
  ) => Promise<any>;
}

export type ExSchemaHandle = {
  enableSubmit: (enable: boolean) => void;
  refresh: () => void;
};

const ExSchema = forwardRef<
  ExSchemaHandle | undefined,
  ExSchemaProps & FormSchema
>((props: ExSchemaProps & FormSchema, ref) => {
  const { onSubmit, ...rest } = props;
  const [changed, setChanged] = useState(false);
  const data = useRef<{ [key: string]: any }>({});
  const [seed, setSeed] = useState(0);

  useImperativeHandle(ref, () => {
    return {
      enableSubmit(enable: boolean) {
        setChanged(enable);
      },
      refresh() {
        setSeed(seed + 1);
      },
    };
  }, [seed, changed]);

  return (
    <BetaSchemaForm
      key={seed}
      submitter={{
        onReset: () => setChanged(false),
        resetButtonProps: { disabled: props.disabled },
        submitButtonProps: { disabled: props.disabled || !changed },
        render: (_, dom) => [...dom.reverse()],
      }}
      // BUG: props.initialvalues is undefined
      onInit={(v) => {
        data.current = v;
        setChanged(false);
      }}
      onValuesChange={(_: any, all: Record<string, any>) => {
        for (let k in all) {
          // possible object
          if (!isEqual(data.current[k], all[k])) {
            setChanged(true);
            return;
          }
        }
        setChanged(false);
      }}
      onFinish={(v) =>
        onSubmit?.(v, data.current).then((rsp) => {
          if (rsp) setSeed(seed + 1);
        })
      }
      {...rest}
    />
  );
});

export default ExSchema;
