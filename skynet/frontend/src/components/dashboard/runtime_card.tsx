import { getAPI, getIntl } from '@/utils';
import { Area } from '@ant-design/charts';
import { StatisticCard } from '@ant-design/pro-components';
import { Col } from 'antd';
import bytes from 'bytes';
import { useEffect, useRef, useState } from 'react';

const RuntimeCard = () => {
  const intl = getIntl();
  const [data, setData] = useState<
    {
      ts: number;
      cpu: number;
      memory: number;
      memory_percent: number;
    }[]
  >([]);
  const tm = useRef<NodeJS.Timeout>();
  const refresh = async () => {
    const msg = await getAPI('/dashboard/runtime_info');
    setData((prev) => [
      ...prev,
      {
        ts: Date.now(),
        cpu: msg.data.cpu,
        memory: msg.data.memory,
        memory_percent: msg.data.memory_percent,
      },
    ]);
  };
  useEffect(() => {
    refresh();
    tm.current = setInterval(refresh, 2000);
    return () => {
      clearInterval(tm.current);
    };
  }, []);

  const cpuConfig = {
    data,
    xField: (d: any) => new Date(d.ts),
    yField: 'cpu',
    axis: {
      x: {
        gridStrokeOpacity: 0.5,
      },
      y: {
        gridStrokeOpacity: 0.5,
      },
    },
    height: 250,
    scale: {
      y: {
        type: 'linear',
        domain: [0, 100],
      },
    },
    style: {
      fill: 'linear-gradient(-90deg, white 0%, cornflowerblue 100%)',
    },
    line: {
      tooltip: false,
      style: {
        stroke: 'cornflowerblue',
        strokeWidth: 2,
      },
    },
    animate: { update: { type: false } },
  };
  const memoryConfig = {
    data,
    xField: (d: any) => new Date(d.ts),
    yField: 'memory_percent',
    axis: {
      x: {
        gridStrokeOpacity: 0.5,
      },
      y: {
        gridStrokeOpacity: 0.5,
      },
    },
    tooltip: {
      items: [
        {
          field: 'memory',
          valueFormatter: (d: number) =>
            bytes.format(d, { unitSeparator: ' ' }) ?? '-',
        },
        {
          field: 'memory_percent',
        },
      ],
    },
    height: 250,
    scale: {
      y: {
        type: 'linear',
        domain: [0, 100],
      },
    },
    style: {
      fill: 'linear-gradient(-90deg, white 0%, cornflowerblue 100%)',
    },
    line: {
      tooltip: false,
      style: {
        stroke: 'cornflowerblue',
        strokeWidth: 2,
      },
    },
    animate: { update: { type: false } },
  };

  return (
    <>
      <Col xs={24} md={12} style={{ height: '300px' }}>
        <StatisticCard
          title={intl.get('pages.dashboard.cpu.title')}
          bordered
          chart={<Area {...cpuConfig} />}
        />{' '}
      </Col>
      <Col xs={24} md={12} style={{ height: '300px' }}>
        <StatisticCard
          title={intl.get('pages.dashboard.memory.title')}
          bordered
          chart={<Area {...memoryConfig} />}
        />{' '}
      </Col>
    </>
  );
};

export default RuntimeCard;
