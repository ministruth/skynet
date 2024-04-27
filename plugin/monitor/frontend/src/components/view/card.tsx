import { getIntl } from '@/utils';
import { ProCard } from '@ant-design/pro-components';
import '@xterm/xterm/css/xterm.css';
import { Tab } from 'rc-tabs/lib/interface';
import { useRef, useState } from 'react';
import DefaultTab from './default';
import ShellTab from './shell';

const ViewCard = () => {
  const intl = getIntl();

  const onChange = (key: string) => {
    setActiveKey(key);
  };
  const onEdit = (e: any, action: 'add' | 'remove') => {
    if (action === 'remove') {
      remove(e);
    }
  };
  const remove = (key: string) => {
    const targetIndex = tabItems.findIndex((x) => x.key === key);
    const newItems = tabItems.filter((x) => x.key !== key);
    if (newItems.length && key === activeKey) {
      const { key } =
        newItems[
          targetIndex === newItems.length ? targetIndex - 1 : targetIndex
        ];
      setActiveKey(key);
    } else if (newItems.length === 0) {
      setActiveKey('default');
    }
    setTabItems(newItems);
  };
  const [activeKey, setActiveKey] = useState('default');
  const [tabItems, setTabItems] = useState<Tab[]>([]);
  const shellIndex = useRef(0);
  return (
    <ProCard
      tabs={{
        type: 'editable-card',
        activeKey: activeKey,
        onChange: onChange,
        hideAdd: true,
        onEdit: onEdit,
        items: [
          {
            key: 'default',
            label: intl.get('pages.view.card.agent'),
            closable: false,
            children: (
              <DefaultTab
                addTabCallback={(row: any) => {
                  const key = `shell${shellIndex.current++}`;
                  setTabItems([
                    ...tabItems,
                    {
                      label: `${row.name}(${shellIndex.current - 1})`,
                      key: key,
                      children: (
                        <ShellTab id={row.id} name={row.name} ip={row.ip} />
                      ),
                    },
                  ]);
                  setActiveKey(key);
                }}
              />
            ),
          },
          ...tabItems,
        ],
      }}
      bordered
    />
  );
};

export default ViewCard;
